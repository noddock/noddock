package nodes

import (
	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {
	param_a := graph.Parameter{
		ID:      "a",
		Kind:    graph.ParamInt,
		Default: 0,
		Connect: []graph.ParamConnect{graph.ConnInput},
	}
	param_b := graph.Parameter{
		ID:      "b",
		Kind:    graph.ParamInt,
		Default: 0,
	}
	param_product := graph.Parameter{
		ID:      "product",
		Kind:    graph.ParamInt,
		Default: 0,
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}
	manager.Register(&graph.NodeSpec{
		ID:          "Core--Calculator--IntMultiply",
		Name:        "Int Multiply",
		Description: "Integer multiplication, product = a * b.",
		Plist: []*graph.Parameter{
			&param_a,
			&param_b,
			&param_product,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			a := ctx.Get(param_a.ID).(int)
			b := ctx.Get(param_b.ID).(int)

			ctx.Set(param_product.ID, a*b)
			return nil
		},
	})
}
