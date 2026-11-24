package nodes

import (
	"github.com/google/uuid"
	"github.com/noddock/noddock/runtime/eventbus"
	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {
	param_audio_in := graph.Parameter{
		ID:      "audio_in",
		Kind:    graph.ParamStream,
		Connect: []graph.ParamConnect{graph.ConnInput},
	}
	param_format := graph.Parameter{
		ID:      "format",
		Kind:    graph.ParamString,
		Default: "audio/wav",
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Audio--AudioStreamViewer",
		Name:        "Audio Stream Viewer",
		Description: `Preview an audio stream in the sandbox window. \n Requires a PCM WAV stream. \n \n The stream is provided to HTML via a Wails binary Stream (wails.Stream("audio")), since raw data is []byte and the event bus / SSE transport text (base64) which is a bad trade.`,
		Plist: []*graph.Parameter{
			&param_audio_in,
			&param_format,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			in := ctx.Get(param_audio_in.ID).(*graph.Stream[[]byte])
			if !in.IsOpen() {
				return nil
			}

			// 1. 建全局二进制通道并注册（StreamAudioHandler 按 key 读取，经 wails.Stream("audio") 转发）
			key := uuid.NewString()
			binCh := make(chan []byte, 64)
			mime := ctx.Get(param_format.ID).(string)
			eventbus.AudioBus.Register(key, &eventbus.AudioStream{Ch: binCh, Mime: mime})
			defer eventbus.AudioBus.Delete(key)
			defer close(binCh)

			// 2. 订阅上游二进制流并 Ready（通知生产者可以推流）
			inCh := in.Subscribe()
			in.Ready()

			// 3. 通知前端：全局音频通道已注册，可开 wails.Stream("audio") 发 key 拉流。
			//    nid = exec id（匹配前端 {{.ID}}），用于多 sandbox 时精确过滤本节点。
			eventbus.Emit(nRT.GetPID(), eventbus.Event{
				Type:    eventbus.EventSseBuildAudio,
				Payload: map[string]any{"key": key, "nid": nRT.GetNID()},
			})

			// 4. 泵：上游 chunk 原封不动转发到全局通道（不做任何变换/分段）
			for {
				select {
				case chunk, ok := <-inCh:
					if !ok {
						return nil // 流关闭
					}
					select {
					case binCh <- chunk:
					case <-nRT.GetRtCtx().Done():
						return nRT.GetRtCtx().Err()
					}
				case <-nRT.GetRtCtx().Done():
					return nRT.GetRtCtx().Err()
				}
			}
		},
		ExecProperty: &graph.ExecProperty{
			CustomRenderPath: "/assets/templates/audio_stream_viewer.html",
		},
	})
}
