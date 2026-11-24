package nodes

import (
	"encoding/json"

	"github.com/noddock/noddock/runtime/eventbus"
	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {
	param_payload := graph.Parameter{
		ID:      "payload",
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
		ID:          "UI--HtmlRequest",
		Name:        "HTML Request",
		Description: `Sends a request to all sandbox and wait for response, each request can only be handled once. In Sandbox Html requires a handler: window.__htmlReqHandler = async (payload) => { return { result: "ok" }}; `,
		Plist: []*graph.Parameter{
			&param_payload,
			&param_result,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			callJson, _ := ctx.Get(param_payload.ID).(json.RawMessage)
			response, err := eventbus.HtmlReqBus.Request(nRT.GetRtCtx(), eventbus.HtmlReq{
				PID:     nRT.GetPID(), // for sending sse events
				Payload: callJson,
			})
			if err != nil {
				return err
			}
			ctx.Set(param_result.ID, json.RawMessage(response.Payload))
			return nil
		},
	})
}
