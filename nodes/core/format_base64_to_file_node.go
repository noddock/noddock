package nodes

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
	"github.com/noddock/noddock/runtime/tool/fo"
)

func init() {
	param_data := graph.Parameter{
		ID:      "data",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}
	param_name_hint := graph.Parameter{
		ID:      "name_hint",
		Name:    "name hint",
		Kind:    graph.ParamString,
		Default: "",
	}
	param_out_dir := graph.Parameter{
		ID:      "out_dir",
		Name:    "out dir",
		Kind:    graph.ParamString,
		Default: ".tmp",
	}
	param_path := graph.Parameter{
		ID:      "path",
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
		ID:          "Core--Format--Base64ToFile",
		Name:        "Base64 to File",
		Description: `Decodes a base64 string into a file. Media (image/audio) get media info; other files are written as .bin. \n out_dir accepts absolute path.`,
		Plist: []*graph.Parameter{
			&param_data,
			&param_name_hint,
			&param_out_dir,
			&param_path,
			&param_info,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			data, _ := ctx.Get(param_data.ID).(string)
			nameHint, _ := ctx.Get(param_name_hint.ID).(string)
			outDir, _ := ctx.Get(param_out_dir.ID).(string)

			raw, err := decodeBase64Loose(data)
			if err != nil {
				return err
			}

			if outDir == "" {
				outDir = "./"
			}
			abs, err := fo.GetPath(outDir, nRT.GetPID(), true, false)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(abs, 0o755); err != nil {
				return err
			}

			name := sanitizeNameHint(nameHint)
			if name == "" {
				name = "file"
			}

			// 识别到媒体用其扩展名，识别不到（通用文件）用 .bin
			ext := ".bin"
			if info, ok := sniffMedia(raw); ok {
				ext = info.Ext
			}
			filename := name + ext
			if err := os.WriteFile(filepath.Join(abs, filename), raw, 0o644); err != nil {
				return err
			}

			// dir 恒为绝对路径；outPath 保持 out_dir 的形态（相对配置→相对路径，不把本机绝对路径带进上下文）
			outPath := filepath.Join(abs, filename)
			if !filepath.IsAbs(outDir) {
				outPath = filepath.Join(outDir, filename)
			}
			ctx.Set(param_path.ID, filepath.ToSlash(outPath))

			b, err := json.Marshal(buildFileInfo(outPath, raw))
			if err != nil {
				return err
			}
			ctx.Set(param_info.ID, json.RawMessage(b))
			return nil
		},
	})
}
