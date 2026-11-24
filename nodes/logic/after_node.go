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

	manager.Register(&graph.NodeSpec{
		ID:          "Logic--After",
		Name:        "After",
		Description: "Logic relay; when the logic input then is activated",
		Plist: []*graph.Parameter{
			&param_then,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			ctx.Set(param_then.ID, "start")
			return nil
		},
	})
}
