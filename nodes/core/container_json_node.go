package nodes

import (
	"encoding/json"
	"errors"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {
	param_json := graph.Parameter{
		ID:      "json",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnInput, graph.ConnEdit, graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Core--Container--Json",
		Name:        "Json",
		Description: "A json container, {} as default, also accept [].",
		Plist: []*graph.Parameter{
			&param_json,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			v, ok := ctx.Get(param_json.ID).(json.RawMessage)
			if !ok {
				return errors.New("input is not jsonRawMessage type")
			}
			ctx.Set(param_json.ID, v)
			return nil
		},
	})
}
