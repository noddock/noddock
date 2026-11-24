package nodes

import (
	"errors"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {
	param_bool := graph.Parameter{
		ID:      "bool",
		Kind:    graph.ParamBool,
		Default: false,
		Connect: []graph.ParamConnect{graph.ConnEdit, graph.ConnInput, graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Core--Container--Bool",
		Name:        "Bool",
		Description: "A boolean container, false as default.",
		Plist: []*graph.Parameter{
			&param_bool,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			v, ok := ctx.Get(param_bool.ID).(bool)
			if !ok {
				return errors.New("input is not bool type")
			}
			ctx.Set(param_bool.ID, v)
			return nil
		},
	})
}
