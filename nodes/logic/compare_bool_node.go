package nodes

import (
	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {
	param_input := graph.Parameter{
		ID:      "bool",
		Kind:    graph.ParamBool,
		Default: false,
		Connect: []graph.ParamConnect{graph.ConnInput},
	}

	param_true := graph.Parameter{
		ID:      "true",
		Kind:    graph.ParamLogic,
		Default: "inactivate",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	param_false := graph.Parameter{
		ID:      "false",
		Kind:    graph.ParamLogic,
		Default: "inactivate",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Logic--Compare--CompareBool",
		Name:        "Compare Bool",
		Description: "Boolean branch.",
		Plist: []*graph.Parameter{
			&param_input,
			&param_true,
			&param_false,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			input := ctx.Get(param_input.ID).(bool)
			if input {
				ctx.Set(param_true.ID, "activate")
			} else {
				ctx.Set(param_false.ID, "activate")

			}
			return nil
		},
	})
}
