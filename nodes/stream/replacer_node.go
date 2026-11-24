package nodes

import (
	"errors"
	"strings"

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
	param_match := graph.Parameter{
		ID:      "match",
		Kind:    graph.ParamString,
		Default: "{user}",
	}
	param_replace := graph.Parameter{
		ID:      "replace",
		Kind:    graph.ParamString,
		Default: "Li Hua",
	}
	manager.Register(&graph.NodeSpec{
		ID:          "Stream--Replacer",
		Name:        "Stream Replacer",
		Description: `Replace match to replace from stream_in to stream_out.`,
		Plist: []*graph.Parameter{
			&param_stream_in,
			&param_stream_out,
			&param_match,
			&param_replace,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			match := ctx.Get(param_match.ID).(string)
			replace := ctx.Get(param_replace.ID).(string)
			if match == "" {
				return errors.New("match must not be empty")
			}

			in := ctx.Get(param_stream_in.ID).(*graph.Stream[string])
			streamNum := nRT.DownstreamCount(param_stream_out.ID)
			out := graph.NewStreamLarge[string](128)
			out.Open()
			out.SetExpectedReady(streamNum)
			defer out.Close()
			ctx.Set(param_stream_out.ID, out)

			if in.IsOpen() {
				if err := out.WaitReady(nRT.GetRtCtx()); err != nil {
					return err
				}
				pending := ""
				inCh := in.Subscribe()
				in.Ready()
			Loop:
				for {
					select {
					case delta, ok := <-inCh:
						if !ok {
							if pending != "" {
								out.Send(pending)
							}
							out.Close()
							break Loop
						}

						pending += delta
						// 处理 pending 中的完整 match 和尾部前缀
						for {
							idx := strings.Index(pending, match)
							if idx >= 0 {
								// 找到完整 match，发送其前面的安全部分
								if idx > 0 {
									out.Send(pending[:idx])
								}
								// 发送替换字符串
								out.Send(replace)
								// 跳过已处理的 match 部分
								pending = pending[idx+len(match):]
								continue // 继续查找下一个 match
							}

							// 没有完整 match，检查尾部是否与 match 的前缀匹配
							keepLen := 0
							for k := len(match) - 1; k > 0; k-- {
								if len(pending) >= k && pending[len(pending)-k:] == match[:k] {
									keepLen = k
									break
								}
							}
							// 发送安全部分（保留可能形成 match 前缀的后缀）
							if len(pending) > keepLen {
								out.Send(pending[:len(pending)-keepLen])
							}
							pending = pending[len(pending)-keepLen:]
							break
						}

					case <-nRT.GetRtCtx().Done():
						if pending != "" {
							out.Send(pending)
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
