package nodes

import (
	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {

	param_count := graph.Parameter{
		ID:      "count",
		Kind:    graph.ParamInt,
		Default: -1,
		Connect: []graph.ParamConnect{graph.ConnEdit, graph.ConnInput, graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Logic--Counter",
		Name:        "Counter",
		Description: "Every run increse count by 1 and return.",
		Plist: []*graph.Parameter{
			&param_count,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			counter := ctx.Get(param_count.ID).(int)
			counter++
			ctx.Set(param_count.ID, counter)
			return nil
		},
	})
}
