package nodes

import (
	"strings"
	"time"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {
	param_stream_in := graph.Parameter{
		ID:      "stream_in",
		Kind:    graph.ParamStream,
		Default: graph.NewStream[string](),
		Connect: []graph.ParamConnect{graph.ConnInput},
	}
	param_stream_out := graph.Parameter{
		ID:      "stream_out",
		Kind:    graph.ParamStream,
		Default: graph.NewStream[string](),
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}
	param_interval := graph.Parameter{
		ID:      "interval",
		Kind:    graph.ParamInt,
		Default: 1000,
	}
	manager.Register(&graph.NodeSpec{
		ID:          "Stream--Rator",
		Name:        "Stream Rator",
		Description: `Limit stream rate by interval (1000 ms default).`,
		Plist: []*graph.Parameter{
			&param_stream_in,
			&param_stream_out,
			&param_interval,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			in := ctx.Get(param_stream_in.ID).(*graph.Stream[string])
			streamNum := nRT.DownstreamCount(param_stream_out.ID)
			out := graph.NewStreamLarge[string](128)
			out.Open()
			out.SetExpectedReady(streamNum)
			defer out.Close()
			ctx.Set(param_stream_out.ID, out)

			interval := ctx.Get(param_interval.ID).(int)

			if in.IsOpen() {
				// 先等本节点下游 ready，再通知上游开始推流
				if err := out.WaitReady(nRT.GetRtCtx()); err != nil {
					return err
				}
				ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
				defer ticker.Stop()

				var buf strings.Builder
				inCh := in.Subscribe()
				in.Ready() // 通知上游生产者：本节点已就绪
			Loop:
				for {
					select {
					case delta, ok := <-inCh: // 监听上游 stream
						if !ok {
							if buf.Len() > 0 {
								out.Send(buf.String())
							}
							out.Close()
							break Loop
						}
						buf.WriteString(delta)

					case <-ticker.C:
						if buf.Len() > 0 {
							out.Send(buf.String())
							buf.Reset()
						}

					case <-nRT.GetRtCtx().Done(): // 可中断
						if buf.Len() > 0 {
							out.Send(buf.String())
						}
						out.Close()
						break Loop
					}
				}
			}
			return nRT.GetRtCtx().Err()
		},
	})
}
