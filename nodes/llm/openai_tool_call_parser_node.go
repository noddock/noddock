package nodes

import (
	"encoding/json"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {
	param_call := graph.Parameter{
		ID:      "call",
		Name:    "tool call",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}
	param_name := graph.Parameter{
		ID:      "name",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}
	param_argument := graph.Parameter{
		ID:      "argument",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}
	param_id := graph.Parameter{
		ID:      "id",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "LLM--OpenAI--ToolCallParser",
		Name:        "Tool Call Parser",
		Description: `Parse a single OpenAI tool call (from tool_calls list) into name (string), argument (JSON object), and id (string).`,
		Plist: []*graph.Parameter{
			&param_call,
			&param_name,
			&param_argument,
			&param_id,
		},
		Run: func(ctx *graph.Context, _ graph.NodeRunTool) error {
			callJson, _ := ctx.Get(param_call.ID).(json.RawMessage)
			tc, err := graph.ParseToolCall(callJson)
			if err != nil {
				return err
			}
			ctx.Set(param_name.ID, tc.Function.Name)
			ctx.Set(param_id.ID, tc.ID)

			args := tc.Args()
			out, err := json.Marshal(args)
			if err != nil {
				return err
			}
			ctx.Set(param_argument.ID, json.RawMessage(out))
			return nil
		},
	})
}
