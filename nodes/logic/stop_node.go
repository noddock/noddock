package nodes

import (
	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {
	param_then := graph.Parameter{
		ID:      "then",
		Kind:    graph.ParamLogic,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnInput, graph.ConnOutput},
	}
	param_stop := graph.Parameter{
		ID:      "stop",
		Kind:    graph.ParamBool,
		Default: true,
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Logic--Stop",
		Name:        "Stop",
		Description: "Will not run on stop equals true",
		Plist: []*graph.Parameter{
			&param_then,
			&param_stop,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			stop, _ := ctx.Get(param_stop.ID).(bool)
			if !stop {
				ctx.Set(param_then.ID, "start")
			}
			return nil
		},
	})
}
