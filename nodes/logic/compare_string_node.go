package nodes

import (
	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {
	param_a := graph.Parameter{
		ID:      "a",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}
	param_b := graph.Parameter{
		ID:      "b",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}

	param_true := graph.Parameter{
		ID:      "a=b",
		Name:    "a==b",
		Kind:    graph.ParamLogic,
		Default: "inactivate",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	param_false := graph.Parameter{
		ID:      "a!=b",
		Name:    "a!=b",
		Kind:    graph.ParamLogic,
		Default: "inactivate",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Logic--Compare--CompareString",
		Name:        "Compare String",
		Description: "String comparison",
		Plist: []*graph.Parameter{
			&param_a,
			&param_b,
			&param_true,
			&param_false,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			a := ctx.Get(param_a.ID).(string)
			b := ctx.Get(param_b.ID).(string)
			if a == b {
				ctx.Set(param_true.ID, "activate")
			} else {
				ctx.Set(param_false.ID, "activate")
			}
			return nil
		},
	})
}
