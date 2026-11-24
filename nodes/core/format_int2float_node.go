package nodes

import (
	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {
	param_int := graph.Parameter{
		ID:      "int",
		Kind:    graph.ParamInt,
		Default: 0,
		Connect: []graph.ParamConnect{graph.ConnInput},
	}

	param_float := graph.Parameter{
		ID:      "float",
		Kind:    graph.ParamFloat,
		Default: float64(0),
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Core--Format--Int2Float",
		Name:        "Int to Float",
		Description: "Converts an int to a float.",
		Plist: []*graph.Parameter{
			&param_int,
			&param_float,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			i := ctx.Get(param_int.ID).(int)
			ctx.Set(param_float.ID, float64(i))
			return nil
		},
	})
}
