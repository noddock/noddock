package nodes

import (
	"encoding/json"
	"fmt"

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
	param_json := graph.Parameter{
		ID:      "json",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}
	manager.Register(&graph.NodeSpec{
		ID:          "Core--Format--String2Json",
		Name:        "String to Json",
		Description: "Validates a string as JSON and passes it through.",
		Plist: []*graph.Parameter{
			&param_string,
			&param_json,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			s := ctx.Get(param_string.ID).(string)
			if !json.Valid([]byte(s)) {
				return fmt.Errorf("invalid json string")
			}
			ctx.Set(param_json.ID, json.RawMessage(s))
			return nil
		},
	})
}
