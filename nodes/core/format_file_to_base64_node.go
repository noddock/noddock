package nodes

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
	"github.com/noddock/noddock/runtime/tool/fo"
)

func init() {
	param_path := graph.Parameter{
		ID:      "path",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}
	param_data := graph.Parameter{
		ID:      "data",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}
	param_info := graph.Parameter{
		ID:      "info",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Core--Format--FileToBase64",
		Name:        "File to Base64",
		Description: `Reads a file from path and converts it to base64. Media (image/audio) get media info; other files get bytes only. \n info : {path, bytes, media_type?, format?, width?, height?}. \n path accepts absolute path, or relative path under this project. \n If the file does not exist, data/info return empty (no error); use Core--File--Exist to check existence.`,
		Plist: []*graph.Parameter{
			&param_path,
			&param_data,
			&param_info,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			path, _ := ctx.Get(param_path.ID).(string)
			path = strings.TrimSpace(path)

			// 空路径 → 返回空值（不报错）
			if path == "" {
				ctx.Set(param_data.ID, "")
				ctx.Set(param_path.ID, path)
				ctx.Set(param_info.ID, json.RawMessage(`{}`))
				return nil
			}

			// 严格解析（allowSymlink=false：解析符号链接防逃逸 + 要求存在）。
			abs, err := fo.GetPath(path, nRT.GetPID(), true, false)
			if err != nil {
				// 区分"路径不存在"（返回空值，不中断）和"路径穿越/空路径"（报错）。
				// 用宽松解析（allowSymlink=true，不要求存在）再判一次：仍失败=真穿越。
				if _, err2 := fo.GetPath(path, nRT.GetPID(), true, true); err2 != nil {
					return err2
				}
				// 路径合法但文件不存在 → data/info 返回空值
				ctx.Set(param_data.ID, "")
				ctx.Set(param_path.ID, path)
				ctx.Set(param_info.ID, json.RawMessage(`{}`))
				return nil
			}
			raw, err := os.ReadFile(abs)
			if err != nil {
				return fmt.Errorf("FileToBase64: failed to read %s: %w", path, err)
			}

			ctx.Set(param_data.ID, base64.StdEncoding.EncodeToString(raw))

			b, err := json.Marshal(buildFileInfo(path, raw))
			if err != nil {
				return fmt.Errorf("FileToBase64: failed to marshal: %w", err)
			}
			ctx.Set(param_path.ID, path)
			ctx.Set(param_info.ID, json.RawMessage(b))
			return nil
		},
	})
}
