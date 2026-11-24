package nodes

import (
	"errors"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {
	param_float := graph.Parameter{
		ID:      "float",
		Kind:    graph.ParamFloat,
		Default: float64(0),
		Connect: []graph.ParamConnect{graph.ConnInput, graph.ConnEdit, graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Core--Container--Float",
		Name:        "Float",
		Description: "A float container, 0.0 as default.",
		Plist: []*graph.Parameter{
			&param_float,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			v, ok := ctx.Get(param_float.ID).(float64)
			if !ok {
				return errors.New("input is not float type")
			}
			ctx.Set(param_float.ID, v)
			return nil
		},
	})
}
