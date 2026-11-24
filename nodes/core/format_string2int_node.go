package nodes

import (
	"strconv"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {

	param_string := graph.Parameter{
		ID:      "string",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}
	param_int := graph.Parameter{
		ID:      "int",
		Kind:    graph.ParamInt,
		Default: 0,
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Core--Format--String2Int",
		Name:        "String to Int",
		Description: "Parses a string as a decimal int.",
		Plist: []*graph.Parameter{
			&param_string,
			&param_int,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			s := ctx.Get(param_string.ID).(string)
			i, err := strconv.Atoi(s)
			if err != nil {
				return err
			}
			ctx.Set(param_int.ID, i)
			return nil
		},
	})
}
