package nodes

import (
	"encoding/json"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {

	param_list := graph.Parameter{
		ID:      "list",
		Kind:    graph.ParamJSON,
		Default: "[]",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}

	param_length := graph.Parameter{
		ID:      "length",
		Kind:    graph.ParamInt,
		Default: 0,
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Json--JsonLength",
		Name:        "Json Length",
		Description: `Calculates the length of a JSON array.`,
		Plist: []*graph.Parameter{
			&param_list,
			&param_length,
		},

		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			listRaw := ctx.Get(param_list.ID).(json.RawMessage)
			var list []any
			if err := json.Unmarshal(listRaw, &list); err != nil {
				return err
			}
			ctx.Set(param_length.ID, len(list))
			return nil
		},
	})
}
