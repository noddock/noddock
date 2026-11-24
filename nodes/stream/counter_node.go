package nodes

import (
	"fmt"
	"time"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {

	param_count := graph.Parameter{
		ID:      "count",
		Kind:    graph.ParamInt,
		Default: 10,
		Connect: []graph.ParamConnect{graph.ConnEdit},
	}
	param_stream_out := graph.Parameter{
		ID:      "stream_out",
		Kind:    graph.ParamStream,
		Default: graph.NewStream[string](),
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}
	manager.Register(&graph.NodeSpec{
		ID:          "Stream--Counter",
		Name:        "Stream Counter",
		Description: "Stream 0 to count every second.",
		Plist: []*graph.Parameter{
			&param_count,
			&param_stream_out,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			counter := ctx.Get(param_count.ID).(int)
			streamNum := nRT.DownstreamCount(param_stream_out.ID)
			out := graph.NewStream[string]()
			out.Open()
			out.SetExpectedReady(streamNum)
			defer out.Close()

			ctx.Set(param_stream_out.ID, out)
			// 等待全部下游消费者 ready 后再开始计数推流
			if err := out.WaitReady(nRT.GetRtCtx()); err != nil {
				return err
			}
			for i := range counter {
				select {
				case <-nRT.GetRtCtx().Done():
					return nRT.GetRtCtx().Err()
				default:
				}
				time.Sleep(time.Second)
				out.Send(fmt.Sprintf("%d", i))
			}
			return nil
		},
	})
}
