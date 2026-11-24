package nodes

import (
	"encoding/json"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {
	param_id := graph.Parameter{
		ID:      "id",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}
	param_content := graph.Parameter{
		ID:      "content",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}
	param_result := graph.Parameter{
		ID:      "result",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "LLM--OpenAI--ToolResultBuilder",
		Name:        "Tool Result Builder",
		Description: `Build a tool result message: {"role":"tool","content":<content>,"tool_call_id":<id>}.`,
		Plist: []*graph.Parameter{
			&param_id,
			&param_content,
			&param_result,
		},
		Run: func(ctx *graph.Context, _ graph.NodeRunTool) error {
			id, _ := ctx.Get(param_id.ID).(string)
			content, _ := ctx.Get(param_content.ID).(json.RawMessage)

			if id == "" {
				return nil
			}
			if len(content) == 0 {
				content = json.RawMessage("{}")
			}

			msg := PayMessage{
				Role:       "tool",
				Content:    string(content),
				ToolCallID: id,
			}
			out, err := json.Marshal(msg)
			if err != nil {
				return err
			}
			ctx.Set(param_result.ID, json.RawMessage(out))
			return nil
		},
	})
}
