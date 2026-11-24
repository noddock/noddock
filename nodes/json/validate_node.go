package nodes

import (
	"encoding/json"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
	"github.com/noddock/noddock/runtime/tool"
)

func init() {
	param_json := graph.Parameter{
		ID:      "json",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}

	param_schema := graph.Parameter{
		ID:      "schema",
		Kind:    graph.ParamJSON,
		Default: "{}",
	}

	param_ok := graph.Parameter{
		ID:      "ok",
		Kind:    graph.ParamBool,
		Default: false,
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Json--Validate",
		Name:        "Json Validate",
		Description: `JSON Schema validation. The schema is cached by its content hash, so it is compiled once and reused across runs.`,
		Plist: []*graph.Parameter{
			&param_json,
			&param_schema,
			&param_ok,
		},

		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			input := ctx.Get(param_json.ID).(json.RawMessage)
			schema := ctx.Get(param_schema.ID).(json.RawMessage)

			var ok bool
			if err := tool.Validate(schema, input); err == nil {
				ok = true
			}

			ctx.Set(param_ok.ID, ok)
			return nil
		},
	})
}
