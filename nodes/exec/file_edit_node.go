package nodes

import (
	"errors"
	"fmt"
	"os"
	"strings"

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
	paramOld := graph.Parameter{
		ID:      "old_string",
		Name:    "old string",
		Kind:    graph.ParamString,
		Connect: []graph.ParamConnect{graph.ConnInput},
		Default: "",
	}
	paramNew := graph.Parameter{
		ID:      "new_string",
		Name:    "new string",
		Kind:    graph.ParamString,
		Connect: []graph.ParamConnect{graph.ConnInput},
		Default: "",
	}
	manager.Register(&graph.NodeSpec{
		ID:                 "Exec--File--Edit",
		RequiredPermission: graph.PermissionDangerous,
		Name:               "File Edit",
		Description:        `Replaces the first occurrence of old_string with new_string in a file (project-relative path). \n Requires project permission >= dangerous. \n Error if old_string not found or appears more than once.`,
		Plist:              []*graph.Parameter{&paramPath, &paramOld, &paramNew},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			defer ctx.Set(paramPath.ID, "")
			path, _ := ctx.Get(paramPath.ID).(string)
			oldStr, _ := ctx.Get(paramOld.ID).(string)
			newStr, _ := ctx.Get(paramNew.ID).(string)
			if oldStr == "" {
				return errors.New("old_string is empty")
			}
			abs, err := resolveProjectPath(nRT, path, true)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(abs)
			if err != nil {
				return fmt.Errorf("read %q: %w", path, err)
			}
			content := string(data)
			count := strings.Count(content, oldStr)
			if count == 0 {
				return fmt.Errorf("old_string not found in %q", path)
			}
			if count > 1 {
				return fmt.Errorf("old_string appears %d times in %q (must be unique)", count, path)
			}
			content = strings.Replace(content, oldStr, newStr, 1)
			if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
				return fmt.Errorf("write %q: %w", path, err)
			}
			return nil
		},
	})
}
