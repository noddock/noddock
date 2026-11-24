package nodes

import (
	"encoding/base64"
	"errors"
	"path/filepath"
	"strings"
)

// Base64ToFile / FileToBase64（通用 base64↔文件 adapter）共用：魔数识别、尺寸解析、宽松 base64 解码、文件名清洗。

// mediaInfo 描述一段字节流是什么媒体（图片或音频）。
type mediaInfo struct {
	MediaType string // image/png / audio/wav ...
	Format    string // png / jpeg / gif / webp / bmp / wav / mp3 ...
	Ext       string // .png
	Width     int    // 音频或解析不出来时为 0
	Height    int
}

// sniffMedia 按魔数识别图片/音频格式并尽量解析宽高；不是已知媒体格式返回 false。
// Accept: png, jpg, jpeg, webp, wav, bmp, mp3, m4a
func sniffMedia(b []byte) (mediaInfo, bool) {
	switch {
	case len(b) >= 24 && string(b[0:8]) == "\x89PNG\r\n\x1a\n":
		return mediaInfo{"image/png", "png", ".png", int(be32(b[16:20])), int(be32(b[20:24]))}, true

	case len(b) >= 3 && b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF:
		w, h := jpegSize(b)
		return mediaInfo{"image/jpeg", "jpeg", ".jpg", w, h}, true

	case len(b) >= 10 && (string(b[0:6]) == "GIF87a" || string(b[0:6]) == "GIF89a"):
		return mediaInfo{
			"image/gif", "gif", ".gif",
			int(b[6]) | int(b[7])<<8,
			int(b[8]) | int(b[9])<<8,
		}, true

	case len(b) >= 12 && string(b[0:4]) == "RIFF" && string(b[8:12]) == "WEBP":
		// WebP 有 VP8 / VP8L / VP8X 三种子格式，尺寸位置各不相同，这里不解析
		return mediaInfo{"image/webp", "webp", ".webp", 0, 0}, true

	case len(b) >= 12 && string(b[0:4]) == "RIFF" && string(b[8:12]) == "WAVE":
		return mediaInfo{"audio/wav", "wav", ".wav", 0, 0}, true

	case len(b) >= 26 && b[0] == 'B' && b[1] == 'M':
		w := int(int32(le32(b[18:22])))
		h := int(int32(le32(b[22:26])))
		if h < 0 {
			h = -h // 自顶向下存储的 BMP
		}
		return mediaInfo{"image/bmp", "bmp", ".bmp", w, h}, true

	// ── 音频 ──
	case len(b) >= 3 && b[0] == 'I' && b[1] == 'D' && b[2] == '3':
		return mediaInfo{"audio/mpeg", "mp3", ".mp3", 0, 0}, true // ID3 头

	case len(b) >= 2 && b[0] == 0xFF && (b[1]&0xE0) == 0xE0:
		return mediaInfo{"audio/mpeg", "mp3", ".mp3", 0, 0}, true // 裸 MPEG 帧同步

	case len(b) >= 4 && string(b[0:4]) == "OggS":
		return mediaInfo{"audio/ogg", "ogg", ".ogg", 0, 0}, true

	case len(b) >= 4 && string(b[0:4]) == "fLaC":
		return mediaInfo{"audio/flac", "flac", ".flac", 0, 0}, true

	case len(b) >= 12 && string(b[4:8]) == "ftyp":
		// m4a/aac 与 mp4 视频同壳（ftyp），这里按音频认（TTS 场景）
		return mediaInfo{"audio/mp4", "m4a", ".m4a", 0, 0}, true
	}
	return mediaInfo{}, false
}

func be32(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

func le32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

// jpegSize 扫 SOF 段取宽高（JPEG 没有固定偏移，必须走标记）。
func jpegSize(b []byte) (int, int) {
	i := 2
	for i+9 < len(b) {
		if b[i] != 0xFF {
			i++
			continue
		}
		marker := b[i+1]
		if marker == 0x01 || (marker >= 0xD0 && marker <= 0xD9) {
			i += 2 // 无长度字段的标记
			continue
		}
		segLen := int(b[i+2])<<8 | int(b[i+3])
		if segLen < 2 {
			return 0, 0
		}
		isSOF := marker >= 0xC0 && marker <= 0xCF &&
			marker != 0xC4 && marker != 0xC8 && marker != 0xCC
		if isSOF {
			h := int(b[i+5])<<8 | int(b[i+6])
			w := int(b[i+7])<<8 | int(b[i+8])
			return w, h
		}
		i += 2 + segLen
	}
	return 0, 0
}

// decodeBase64Loose 宽松解码：剥掉 data URL 前缀、去掉空白、容忍缺失的 padding，
func decodeBase64Loose(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "data:") {
		if i := strings.Index(s, "base64,"); i >= 0 {
			s = s[i+len("base64,"):]
		}
	}

	s = strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '\r':
			return -1
		}
		return r
	}, s)
	if s == "" {
		return nil, errors.New("base64 data is empty")
	}

	for _, enc := range []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	} {
		if b, err := enc.DecodeString(s); err == nil {
			return b, nil
		}
	}
	return nil, errors.New("not valid base64 data")
}

func sanitizeNameHint(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == '/' || r == '\\' || r == '.' || r == ' ':
			b.WriteRune('-')
		case r > 127:
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), "-")
	if r := []rune(out); len(r) > 64 {
		out = string(r[:64])
	}
	return out
}

// buildFileInfo 构建 info 输出：识别到媒体→带 media_type/format/width/height；识别不到（通用文件）→只带 bytes。
// path 用调用方传入的形态（相对/绝对），内部统一 ToSlash。
func buildFileInfo(path string, raw []byte) map[string]any {
	out := map[string]any{
		"path":  filepath.ToSlash(path),
		"bytes": len(raw),
	}
	if info, ok := sniffMedia(raw); ok {
		out["media_type"] = info.MediaType
		out["format"] = info.Format
		if info.Width > 0 && info.Height > 0 {
			out["width"] = info.Width
			out["height"] = info.Height
		}
	}
	return out
}
