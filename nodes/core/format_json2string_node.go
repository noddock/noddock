package nodes

import (
	"encoding/json"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {
	param_json := graph.Parameter{
		ID:      "json",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}

	param_string := graph.Parameter{
		ID:      "string",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Core--Format--Json2String",
		Name:        "Json to String",
		Description: "Converts a JSON value to a string in compact form, no indentation.",
		Plist: []*graph.Parameter{
			&param_json,
			&param_string,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			json := ctx.Get(param_json.ID).(json.RawMessage)
			ctx.Set(param_string.ID, string(json))
			return nil
		},
	})
}
