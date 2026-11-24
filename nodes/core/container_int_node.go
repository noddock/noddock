package nodes

import (
	"errors"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {

	param_int := graph.Parameter{
		ID:      "int",
		Kind:    graph.ParamInt,
		Default: 0,
		Connect: []graph.ParamConnect{graph.ConnInput, graph.ConnEdit, graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Core--Container--Int",
		Name:        "Int",
		Description: "A int container, 0 as default.",
		Plist: []*graph.Parameter{
			&param_int,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			v, ok := ctx.Get(param_int.ID).(int)
			if !ok {
				return errors.New("input is not int type")
			}
			ctx.Set(param_int.ID, v)
			return nil
		},
	})
}
