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
	paramPattern := graph.Parameter{
		ID:      "pattern",
		Kind:    graph.ParamString,
		Connect: []graph.ParamConnect{graph.ConnInput},
		Default: "",
	}
	paramMatches := graph.Parameter{
		ID:      "matches",
		Kind:    graph.ParamString,
		Connect: []graph.ParamConnect{graph.ConnOutput},
		Default: "",
	}
	manager.Register(&graph.NodeSpec{
		ID:                 "Exec--File--Grep",
		RequiredPermission: graph.PermissionDangerous,
		Name:               "File Grep",
		Description:        `Searches a file for lines containing the pattern (substring match, case-sensitive). \n path: relative to project workspace. \n pattern: substring to search for. \n matches: matching lines joined by newline (empty if no match).`,
		Plist:              []*graph.Parameter{&paramPath, &paramPattern, &paramMatches},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			defer ctx.Set(paramPath.ID, "")
			path, _ := ctx.Get(paramPath.ID).(string)
			pattern, _ := ctx.Get(paramPattern.ID).(string)
			if pattern == "" {
				return errors.New("pattern is empty")
			}
			abs, err := resolveProjectPath(nRT, path, true)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(abs)
			if err != nil {
				return fmt.Errorf("read %q: %w", path, err)
			}
			var matched []string
			for i, line := range strings.Split(string(data), "\n") {
				if strings.Contains(line, pattern) {
					matched = append(matched, fmt.Sprintf("%d: %s", i+1, line))
				}
			}
			ctx.Set(paramMatches.ID, strings.Join(matched, "\n"))
			return nil
		},
	})
}
