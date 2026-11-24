package nodes

import (
	"fmt"

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

	param_string := graph.Parameter{
		ID:      "string",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Core--Format--Int2String",
		Name:        "Int to String",
		Description: "Converts an int to a decimal string, negative values keep the - sign.",
		Plist: []*graph.Parameter{
			&param_int,
			&param_string,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			i := ctx.Get(param_int.ID).(int)
			ctx.Set(param_string.ID, fmt.Sprintf("%d", i))
			return nil
		},
	})
}
