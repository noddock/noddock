package nodes

import (
	"fmt"
	"os"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
	"github.com/noddock/noddock/runtime/tool/fo"
)

func resolveProjectPath(nRT graph.NodeRunTool, path string, resolveSymlink bool) (string, error) {
	return fo.GetPath(path, nRT.GetPID(), false, resolveSymlink)
}

func init() {
	paramPath := graph.Parameter{
		ID:      "path",
		Kind:    graph.ParamString,
		Connect: []graph.ParamConnect{graph.ConnInput},
		Default: "",
	}
	paramContent := graph.Parameter{
		ID:      "content",
		Kind:    graph.ParamString,
		Connect: []graph.ParamConnect{graph.ConnOutput},
		Default: "",
	}
	manager.Register(&graph.NodeSpec{
		ID:                 "Exec--File--Read",
		RequiredPermission: graph.PermissionDangerous,
		Name:               "File Read",
		Description:        `Reads a file's content (project-relative path). \n path: relative to project workspace (absolute paths rejected). \n content: full file content as string.`,
		Plist:              []*graph.Parameter{&paramPath, &paramContent},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			defer ctx.Set(paramPath.ID, "")
			path, _ := ctx.Get(paramPath.ID).(string)
			abs, err := resolveProjectPath(nRT, path, true)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(abs)
			if err != nil {
				return fmt.Errorf("read %q: %w", path, err)
			}
			ctx.Set(paramContent.ID, string(data))
			return nil
		},
	})

}
