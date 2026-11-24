package nodes

import (
	"errors"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {
	param_string := graph.Parameter{
		ID:      "string",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnEdit, graph.ConnInput, graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Core--Container--String",
		Name:        "String",
		Description: `A string/text container, empty as default. \n You can click the ID to switch from single line to multiple lines.`,
		Plist: []*graph.Parameter{
			&param_string,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			s, ok := ctx.Get(param_string.ID).(string)
			if !ok {
				return errors.New("input is not string type")
			}
			ctx.Set(param_string.ID, s)
			return nil
		},
	})
}
