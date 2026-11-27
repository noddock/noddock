package nodes

import (
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
	param_exist := graph.Parameter{
		ID:      "exist",
		Kind:    graph.ParamBool,
		Default: false,
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Core--File--Exist",
		Name:        "File Exist",
		Description: `Checks whether a file (or directory) exists. \n path accepts absolute path, or relative path under this project. \n exist: true if the path resolves to an existing file/dir, false otherwise. \n Path traversal (escaping the project) returns an error; a non-existent but valid path returns exist=false.`,
		Plist: []*graph.Parameter{
			&param_path,
			&param_exist,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			path, _ := ctx.Get(param_path.ID).(string)
			path = strings.TrimSpace(path)
			if path == "" {
				ctx.Set(param_exist.ID, false)
				return nil
			}

			// 严格解析（allowSymlink=false：解析符号链接防逃逸 + 要求存在）。
			abs, err := fo.GetPath(path, nRT.GetPID(), true, false)
			if err != nil {
				// 区分"不存在"（exist=false）和"路径穿越/空路径"（报错）。
				if _, err2 := fo.GetPath(path, nRT.GetPID(), true, true); err2 != nil {
					return err2
				}
				ctx.Set(param_exist.ID, false)
				return nil
			}
			// abs 已存在（GetPath allowSymlink=false 要求存在）；再 Stat 确认是文件/目录。
			if _, err := os.Stat(abs); err != nil {
				ctx.Set(param_exist.ID, false)
				return nil
			}
			ctx.Set(param_exist.ID, true)
			return nil
		},
	})
}
