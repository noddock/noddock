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
	param_path := graph.Parameter{
		ID:      "path",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "LLM--AddMedia",
		Name:        "Add Media",
		Description: `Adds a media file path to a message's content as a multimodal content part. \n input: message (single OpenAI message) + path (file path, relative to project or absolute). \n The path is stored as-is (NOT base64); the Chat node resolves it to the workspace and converts to base64 at request time (keeps the conversation small). \n Empty path → no media part added.`,
		Plist: []*graph.Parameter{
			&param_message,
			&param_path,
		},
		Run: func(ctx *graph.Context, _ graph.NodeRunTool) error {
			// 1. 读取输入
			rawMsg, _ := ctx.Get(param_message.ID).(json.RawMessage)
			path := strings.TrimSpace(ctx.Get(param_path.ID).(string))

			// 2. 解析 message（map 保留所有字段，不只 role/content）
			var msg map[string]any
			if err := json.Unmarshal(rawMsg, &msg); err != nil {
				return fmt.Errorf("LLM AddMedia: message must be a JSON object: %w", err)
			}

			// 3. 构建多模态 content 数组：现有 content（string→text part / array→保留）+ media part
			var parts []any
			switch c := msg["content"].(type) {
			case string:
				if c != "" {
					parts = append(parts, map[string]any{"type": "text", "text": c})
				}
			case []any:
				parts = append(parts, c...)
			}

			// 4. path 非空 → 存路径（相对/绝对，非 base64）；Chat 节点发请求时解析 workspace 读文件转 base64
			//    （对话只存路径，体积大幅缩小；conversation.html 也能用相对路径直接引用）
			if path != "" {
				_, isImage := mediaMime(path)
				if isImage {
					parts = append(parts, map[string]any{
						"type":      "image_url",
						"image_url": map[string]any{"url": path},
					})
				} else {
					parts = append(parts, map[string]any{
						"type": "file",
						"file": map[string]any{"filename": filepath.Base(path), "file_data": path},
					})
				}
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
