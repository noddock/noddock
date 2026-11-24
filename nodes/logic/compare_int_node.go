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
		Default: 10,
		Connect: []graph.ParamConnect{graph.ConnInput},
	}

	param_true := graph.Parameter{
		ID:      "a_large_equal_b",
		Name:    "a≥b",
		Kind:    graph.ParamLogic,
		Default: "inactivate",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	param_false := graph.Parameter{
		ID:      "a_less_b",
		Name:    "a<b",
		Kind:    graph.ParamLogic,
		Default: "inactivate",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	param_equal := graph.Parameter{
		ID:      "a_equal_b",
		Name:    "a==b",
		Kind:    graph.ParamLogic,
		Default: "inactivate",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	param_not_equal := graph.Parameter{
		ID:      "a_not_equal_b",
		Name:    "a!=b",
		Kind:    graph.ParamLogic,
		Default: "inactivate",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Logic--Compare--CompareInt",
		Name:        "Compare Int",
		Description: "Integer comparison.",
		Plist: []*graph.Parameter{
			&param_a,
			&param_b,
			&param_true,
			&param_false,
			&param_equal,
			&param_not_equal,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			a := ctx.Get(param_a.ID).(int)
			b := ctx.Get(param_b.ID).(int)
			if a >= b {
				ctx.Set(param_true.ID, "activate")
			} else {
				ctx.Set(param_false.ID, "activate")
			}
			if a == b {
				ctx.Set(param_equal.ID, "activate")
			} else {
				ctx.Set(param_not_equal.ID, "activate")
			}
			return nil
		},
	})
}
