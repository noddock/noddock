package nodes

import (
	"fmt"
	"strings"

	"github.com/noddock/noddock/runtime/eventbus"
	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"

	"github.com/google/uuid"
)

func init() {
	param_stream_input := graph.Parameter{
		ID:      "stream_input",
		Kind:    graph.ParamStream,
		Default: graph.NewStream[string](),
		Connect: []graph.ParamConnect{graph.ConnInput},
	}
	param_stream_output := graph.Parameter{
		ID:      "stream_output",
		Kind:    graph.ParamStream,
		Default: graph.NewStream[string](),
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	param_result := graph.Parameter{
		ID:      "result",
		Kind:    graph.ParamString,
		Default: "",
	}
	manager.Register(&graph.NodeSpec{
		ID:          "Stream--StreamViewer",
		Name:        "Stream Viewer",
		Description: `Display stream text delta.`,
		Plist: []*graph.Parameter{
			&param_stream_input,
			&param_result,
			&param_stream_output,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			in := ctx.Get(param_stream_input.ID).(*graph.Stream[string])
			streamNum := nRT.DownstreamCount(param_stream_output.ID)
			out := graph.NewStream[string]()
			out.Open()
			out.SetExpectedReady(streamNum)
			defer out.Close()

			ctx.Set(param_stream_output.ID, out)
			if in.IsOpen() {
				// 先等本节点下游 ready，再通知上游开始推流
				if err := out.WaitReady(nRT.GetRtCtx()); err != nil {
					return err
				}

				nodeKey := fmt.Sprintf("streamviewer_run_%s", uuid.New())
				nRT.NotifySiblingsNewStream(nodeKey, param_stream_input.ID)
				var full strings.Builder
				inCh := in.Subscribe()
				in.Ready() // 通知上游生产者：本节点已就绪
				for {

					select {
					case delta, ok := <-inCh: // 监听上游 stream
						if !ok {
							eventbus.Emit(nodeKey, eventbus.Event{Type: eventbus.EventSSEClose, Payload: map[string]any{}})
							ctx.Set(param_result.ID, full.String())
							return nil
						}
						full.WriteString(delta)
						out.Send(delta)
						eventbus.Emit(nodeKey, eventbus.Event{Type: eventbus.EventLLMChunk, Delta: delta})
					case <-nRT.GetRtCtx().Done():
						return nRT.GetRtCtx().Err()
					}
				}

			}
			return nil
		},
		ExecProperty: &graph.ExecProperty{
			CustomRenderPath: "/assets/templates/stream_viewer.html",
		},
	})
}
