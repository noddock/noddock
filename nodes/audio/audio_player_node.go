package nodes

import (
	"os"
	"path/filepath"
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
	param_volume := graph.Parameter{
		ID:      "volume",
		Kind:    graph.ParamFloat,
		Default: 0.5,
		Connect: []graph.ParamConnect{graph.ConnEdit},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Audio--AudioPlayer",
		Name:        "Audio Player",
		Description: `Play an audio file in the sandbox window. \n Path must be relative.`,
		Plist: []*graph.Parameter{
			&param_path,
			&param_volume,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			// 0. clean input
			var err error
			defer func() {
				if err != nil {
					ctx.Set(param_path.ID, "")
				}
			}()

			path := ctx.Get(param_path.ID).(string)
			if strings.TrimSpace(path) == "" {
				return nil
			}

			// 1. any path to absolute path
			abs, err := fo.GetPath(path, nRT.GetPID(), false, false)
			if err != nil {
				return err
			}

			// 2. check existence
			if _, err := os.Stat(abs); err != nil {
				return err
			}

			// 3. abs to rel
			rel, err := fo.RelPro(abs, nRT.GetPID())
			if err != nil {
				return err
			}

			// 4. set result
			ctx.Set(param_path.ID, filepath.ToSlash(rel))
			return nil
		},
		ExecProperty: &graph.ExecProperty{
			CustomRenderPath: "/assets/templates/audio_player.html",
		},
	})
}
