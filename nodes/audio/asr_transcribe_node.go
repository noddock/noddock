package nodes

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {
	param_audio := graph.Parameter{
		ID:      "audio",
		Name:    "audio (base64)",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}
	param_path := graph.Parameter{
		ID:      "path",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}
	param_endpoint := graph.Parameter{
		ID:      "endpoint",
		Kind:    graph.ParamString,
		Default: "http://127.0.0.1:8000/v1/audio/transcriptions",
	}
	param_api := graph.Parameter{
		ID:      "api",
		Kind:    graph.ParamString,
		Default: "",
	}
	param_model := graph.Parameter{
		ID:      "model",
		Kind:    graph.ParamString,
		Default: "mlx-community/whisper-large-v3-turbo",
	}
	param_settings_extra := graph.Parameter{
		ID:      "settings_extra",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnHidden},
	}
	param_transcript := graph.Parameter{
		ID:      "transcript",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Audio--ASR--Transcribe",
		Name:        "ASR Transcribe",
		Description: `Transcribe audio via the /v1/audio/transcriptions API (OpenAI-compatible, mlx-audio). \n input: audio (base64, from Core--Format--FileToBase64) + path (file path, for filename/extension). \n output: transcript (string). \n Empty audio returns empty transcript (no error). \n Check details at https://github.com/Neywa-Labs/mlx-audio/blob/main/docs/guides/web-ui-api-server.md`,
		Plist: []*graph.Parameter{
			&param_audio,
			&param_path,
			&param_endpoint,
			&param_api,
			&param_model,
			&param_settings_extra,
			&param_transcript,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			audioB64 := ctx.Get(param_audio.ID).(string)
			path := ctx.Get(param_path.ID).(string)
			api := ctx.Get(param_api.ID).(string)
			model := ctx.Get(param_model.ID).(string)
			endpoint := ctx.Get(param_endpoint.ID).(string)

			if api != "" {
				if resolved, err := nRT.VaultLookup(api); err == nil {
					api = resolved
				}
			}

			// 1. decode base64 audio（空 → 返回空 transcript，不报错）
			audioBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(audioB64))
			if err != nil {
				return fmt.Errorf("ASR: failed to decode base64 audio: %w", err)
			}
			if len(audioBytes) == 0 {
				ctx.Set(param_transcript.ID, "")
				return nil
			}

			// 2. build multipart form（file + model + response_format + settings_extra）
			var buf bytes.Buffer
			w := multipart.NewWriter(&buf)
			filename := filepath.Base(path)
			if filename == "" || filename == "." || filename == string(filepath.Separator) {
				filename = "audio.mp3"
			}
			fw, err := w.CreateFormFile("file", filename)
			if err != nil {
				return fmt.Errorf("ASR: failed to create form file: %w", err)
			}
			if _, err := fw.Write(audioBytes); err != nil {
				return fmt.Errorf("ASR: failed to write audio: %w", err)
			}
			_ = w.WriteField("model", model)
			_ = w.WriteField("response_format", "json")

			// settings_extra：string/bool/number 直接写，复杂值 JSON 序列化
			var extraMap map[string]any
			if err := json.Unmarshal(ctx.Get(param_settings_extra.ID).(json.RawMessage), &extraMap); err == nil {
				for k, v := range extraMap {
					if k == "file" || k == "model" || k == "response_format" {
						continue
					}
					switch val := v.(type) {
					case string:
						_ = w.WriteField(k, val)
					case bool:
						_ = w.WriteField(k, fmt.Sprintf("%v", val))
					case float64:
						_ = w.WriteField(k, fmt.Sprintf("%v", val))
					default:
						b, _ := json.Marshal(val)
						_ = w.WriteField(k, string(b))
					}
				}
			}
			if err := w.Close(); err != nil {
				return err
			}

			// 3. send request
			req, err := http.NewRequestWithContext(nRT.GetRtCtx(), http.MethodPost, endpoint, &buf)
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", w.FormDataContentType())
			req.Header.Set("Authorization", "Bearer "+api)

			client := &http.Client{} // 无超时：ASR 可能较慢，靠 ctx 中断
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
				return fmt.Errorf("mlx-asr: %s: %s", resp.Status, strings.TrimSpace(string(b)))
			}

			// 4. parse response {"text": "..."}（fallback：纯文本）
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			var result struct {
				Text string `json:"text"`
			}
			if err := json.Unmarshal(body, &result); err != nil {
				ctx.Set(param_transcript.ID, strings.TrimSpace(string(body)))
				return nil
			}
			ctx.Set(param_transcript.ID, result.Text)
			return nil
		},
	})
}
