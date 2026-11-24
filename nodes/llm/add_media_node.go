package nodes

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

// mediaMime 根据文件后缀名返回 MIME 类型 + 是否图像。
func mediaMime(path string) (string, bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png", true
	case ".jpg", ".jpeg":
		return "image/jpeg", true
	case ".gif":
		return "image/gif", true
	case ".webp":
		return "image/webp", true
	case ".bmp":
		return "image/bmp", true
	case ".pdf":
		return "application/pdf", false
	case ".txt":
		return "text/plain", false
	case ".csv":
		return "text/csv", false
	default:
		return "application/octet-stream", false
	}
}

func init() {
	param_message := graph.Parameter{
		ID:      "message",
		Kind:    graph.ParamJSON,
		Default: `{"role": "user", "content": "what is this file"}`,
		Connect: []graph.ParamConnect{graph.ConnInput, graph.ConnOutput},
	}
	param_data := graph.Parameter{
		ID:      "data",
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

	manager.Register(&graph.NodeSpec{
		ID:          "LLM--AddMedia",
		Name:        "Add Media",
		Description: `Adds a media file (base64) to a message's content as a multimodal content part. \n input: message (single OpenAI message) + data (base64, from Core--Format--FileToBase64) + path (file path, for extension detection).`,
		Plist: []*graph.Parameter{
			&param_message,
			&param_data,
			&param_path,
		},
		Run: func(ctx *graph.Context, _ graph.NodeRunTool) error {
			// 1. 读取输入
			rawMsg, _ := ctx.Get(param_message.ID).(json.RawMessage)
			data, _ := ctx.Get(param_data.ID).(string)
			path, _ := ctx.Get(param_path.ID).(string)

			// 2. 根据后缀名判断 MIME + 是否图像
			mime, isImage := mediaMime(path)
			filename := filepath.Base(path)

			// 3. 解析 message（map 保留所有字段，不只 role/content）
			var msg map[string]any
			if err := json.Unmarshal(rawMsg, &msg); err != nil {
				return fmt.Errorf("LLM AddMedia: message must be a JSON object: %w", err)
			}

			// 4. 构建多模态 content 数组：现有 content（string→text part / array→保留）+ media part
			var parts []any
			switch c := msg["content"].(type) {
			case string:
				if c != "" {
					parts = append(parts, map[string]any{"type": "text", "text": c})
				}
			case []any:
				parts = append(parts, c...)
			}
			dataURL := fmt.Sprintf("data:%s;base64,%s", mime, data)
			if isImage {
				parts = append(parts, map[string]any{
					"type":      "image_url",
					"image_url": map[string]any{"url": dataURL},
				})
			} else {
				parts = append(parts, map[string]any{
					"type": "file",
					"file": map[string]any{"filename": filename, "file_data": dataURL},
				})
			}

			// 5. 写回整合后的 message（content 变多模态数组，其他字段保留）
			msg["content"] = parts
			outJson, err := json.Marshal(msg)
			if err != nil {
				return err
			}
			ctx.Set(param_message.ID, json.RawMessage(outJson))
			return nil
		},
	})
}
