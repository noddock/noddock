package nodes

import (
	"encoding/json"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func registerTypedAdder(nodeID, name string, kind graph.ParamKind, defaultValue any) {
	param_json := graph.Parameter{
		ID:      "json",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnInput, graph.ConnOutput},
	}
	param_key := graph.Parameter{
		ID:      "key",
		Kind:    graph.ParamString,
		Default: "",
	}
	param_value := graph.Parameter{
		ID:      "value",
		Kind:    kind,
		Default: defaultValue,
		Connect: []graph.ParamConnect{graph.ConnInput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          nodeID,
		Name:        name,
		Description: `Adds/sets a ` + string(kind) + ` field to a JSON object.`,
		Plist: []*graph.Parameter{
			&param_json,
			&param_key,
			&param_value,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			input := ctx.Get(param_json.ID).(json.RawMessage)
			key := ctx.Get(param_key.ID).(string)
			value := ctx.Get(param_value.ID)

			var js map[string]any
			if err := json.Unmarshal(input, &js); err != nil {
				return err
			}
			if js == nil {
				js = map[string]any{}
			}

			js[key] = value
			v, err := json.Marshal(js)
			if err != nil {
				return err
			}
			ctx.Set(param_json.ID, json.RawMessage(v))
			return nil
		},
	})
}

func init() {
	registerTypedAdder("Json--Property--SetBool", "Set Bool", graph.ParamBool, false)
	registerTypedAdder("Json--Property--SetInt", "Set Int", graph.ParamInt, 0)
	registerTypedAdder("Json--Property--SetFloat", "Set Float", graph.ParamFloat, float64(0))
	registerTypedAdder("Json--Property--SetString", "Set String", graph.ParamString, "")
	registerTypedAdder("Json--Property--SetJson", "Set Json", graph.ParamJSON, json.RawMessage(`{}`))
}
