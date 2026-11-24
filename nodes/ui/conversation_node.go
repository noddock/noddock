package nodes

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/noddock/noddock/runtime/eventbus"
	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {

	param_messages := graph.Parameter{
		ID:   "messages",
		Kind: graph.ParamJSON,
		Default: `[
			{"role": "system", "content": "You are a helpful assistant!"}
		]`,
		Connect: []graph.ParamConnect{graph.ConnInput, graph.ConnOutput},
	}
	param_stream_out := graph.Parameter{
		ID:      "stream_out",
		Kind:    graph.ParamStream,
		Default: graph.NewStream[string](),
		Connect: []graph.ParamConnect{graph.ConnInput},
	}
	param_chat_workflow := graph.Parameter{
		ID:      "chat_workflow",
		Kind:    graph.ParamString,
		Default: "",
	}
	param_compressWorkflow := graph.Parameter{
		ID:      "compress_workflow",
		Kind:    graph.ParamString,
		Default: "",
	}
	manager.Register(&graph.NodeSpec{
		ID:          "UI--Conversation",
		Name:        "Conversation",
		Description: `A conversation UI node: manages messages, triggers run/compress workflows, streams LLM output. \n Compress: the button triggers compress_workflow; the actual processing (add prompt + LLM summary + replace) happens in that workflow.`,
		Plist: []*graph.Parameter{
			&param_messages,
			&param_stream_out,
			&param_chat_workflow,
			&param_compressWorkflow,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			// 1. history 契约校验（必须是 JSON 数组）+ 规范化写回（触发前端 update 同步）
			raw := ctx.Get(param_messages.ID).(json.RawMessage)
			var msgs []any
			if err := json.Unmarshal(raw, &msgs); err != nil {
				return err
			}
			v, _ := json.Marshal(msgs)
			ctx.Set(param_messages.ID, json.RawMessage(v))

			// 2. LLM 流式预览：上游流打开时泵增量到 sandbox，流关闭即结束。
			in := ctx.Get(param_stream_out.ID).(*graph.Stream[string])
			if !in.IsOpen() {
				return nil
			}

			// 4. 通知前端 sandbox 建立 LLM 流式预览连接（经 Wails Stream）
			nodeKey := fmt.Sprintf("chat_stream_%s", uuid.New())
			nRT.NotifySiblingsNewStream(nodeKey, param_stream_out.ID)

			inCh := in.Subscribe()
			in.Ready()

			for {
				select {
				case delta, ok := <-inCh:
					if !ok {
						eventbus.Emit(nodeKey, eventbus.Event{Type: eventbus.EventSSEClose, Payload: map[string]any{}})
						return nil
					}
					eventbus.Emit(nodeKey, eventbus.Event{Type: eventbus.EventLLMChunk, Delta: delta})
				case <-nRT.GetRtCtx().Done():
					return nRT.GetRtCtx().Err()
				}
			}

		},
		ExecProperty: &graph.ExecProperty{
			CustomRenderPath: "/assets/templates/conversation.html",
		},
	})
}
