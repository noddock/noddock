package nodes

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

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
		Connect: []graph.ParamConnect{graph.ConnInput},
		Default: "",
	}
	manager.Register(&graph.NodeSpec{
		ID:                 "Exec--File--Write",
		RequiredPermission: graph.PermissionDangerous,
		Name:               "File Write",
		Description:        `Writes content to a file (project-relative path). Creates parent directories if needed. Overwrites existing file. \n Requires project permission >= dangerous.`,
		Plist:              []*graph.Parameter{&paramPath, &paramContent},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			defer ctx.Set(paramPath.ID, "")
			path, _ := ctx.Get(paramPath.ID).(string)
			content, _ := ctx.Get(paramContent.ID).(string)
			abs, err := resolveProjectPath(nRT, path, false)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				return fmt.Errorf("mkdir for %q: %w", path, err)
			}
			if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
				return fmt.Errorf("write %q: %w", path, err)
			}
			return nil
		},
	})
}
