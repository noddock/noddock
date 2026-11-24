package nodes

import (
	"encoding/json"
	"errors"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func registerTypedDivider(nodeID, name string, kind graph.ParamKind, defaultValue any) {
	param_json := graph.Parameter{
		ID:      "json",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnInput},
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
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          nodeID,
		Name:        name,
		Description: `Extracts a ` + string(kind) + ` field from a JSON object.`,
		Plist: []*graph.Parameter{
			&param_json,
			&param_key,
			&param_value,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			input := ctx.Get(param_json.ID).(json.RawMessage)
			key := ctx.Get(param_key.ID).(string)

			var j map[string]any
			if err := json.Unmarshal(input, &j); err != nil {
				return err
			}
			if j == nil {
				return errors.New(nodeID + ": input must be a JSON object")
			}

			v, ok := j[key]
			if !ok {
				return nil // 字段不存在，保持默认值
			}

			vv, err := param_value.ValidateAndCast(v)
			if err != nil {
				return err
			}
			ctx.Set(param_value.ID, vv)
			return nil
		},
	})
}

func init() {
	registerTypedDivider("Json--Property--GetBool", "Get Bool", graph.ParamBool, false)
	registerTypedDivider("Json--Property--GetInt", "Get Int", graph.ParamInt, 0)
	registerTypedDivider("Json--Property--GetFloat", "Get Float", graph.ParamFloat, float64(0))
	registerTypedDivider("Json--Property--GetString", "Get String", graph.ParamString, "")
	registerTypedDivider("Json--Property--GetJson", "Get Json", graph.ParamJSON, json.RawMessage(`{}`))
}
