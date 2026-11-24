package nodes

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {
	param_text := graph.Parameter{
		ID:      "text",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}
	param_ref_audio := graph.Parameter{
		ID:      "ref_audio",
		Name:    "ref audio (base64)",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}
	param_ref_text := graph.Parameter{
		ID:      "ref_text",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}

	param_endpoint := graph.Parameter{
		ID:      "endpoint",
		Kind:    graph.ParamString,
		Default: "http://127.0.0.1:8000/v1/audio/speech",
	}
	param_api := graph.Parameter{
		ID:      "api",
		Kind:    graph.ParamString,
		Default: "",
	}
	param_model := graph.Parameter{
		ID:      "model",
		Kind:    graph.ParamString,
		Default: "Kokoro-82M-bf16",
	}
	param_settings_extra := graph.Parameter{
		ID:      "settings_extra",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnHidden},
	}

	param_stream_out := graph.Parameter{
		ID:      "stream_out",
		Kind:    graph.ParamStream,
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}
	param_audio_out := graph.Parameter{
		ID:      "audio_out",
		Name:    "audio out (base64)",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Audio--TTS--Speech",
		Name:        "TTS Speech",
		Description: `Synthesize speech via the /v1/audio/speech API (OpenAI-compatible). \n Check details at https://github.com/Neywa-Labs/mlx-audio/blob/main/docs/guides/web-ui-api-server.md`,
		Plist: []*graph.Parameter{
			&param_text,
			&param_ref_audio,
			&param_ref_text,
			&param_endpoint,
			&param_api,
			&param_model,
			&param_settings_extra,
			&param_stream_out,
			&param_audio_out,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			var err error
			defer func() {
				if err != nil {
					ctx.Set(param_ref_audio.ID, "")
					ctx.Set(param_ref_text.ID, "")
				}
			}()

			text := ctx.Get(param_text.ID).(string)
			refAudio := ctx.Get(param_ref_audio.ID).(string)
			refText := ctx.Get(param_ref_text.ID).(string)
			api := ctx.Get(param_api.ID).(string)
			model := ctx.Get(param_model.ID).(string)
			endpoint := ctx.Get(param_endpoint.ID).(string)
			streamNum := nRT.DownstreamCount(param_stream_out.ID)
			streaming := streamNum > 0

			if api != "" {
				if resolved, err := nRT.VaultLookup(api); err == nil {
					api = resolved
				}
			}

			// 1. build body
			body := map[string]any{
				"model":           model,
				"input":           text,
				"response_format": "wav",
			}
			if refAudio != "" {
				body["ref_audio"] = refAudio
			}
			if refText != "" {
				body["ref_text"] = refText
			}
			if streaming {
				body["stream"] = true
				body["streaming_interval"] = 0.2
			}

			var extraMap map[string]any
			if err := json.Unmarshal(ctx.Get(param_settings_extra.ID).(json.RawMessage), &extraMap); err == nil {
				for k, v := range extraMap {
					body[k] = v
				}
			}

			// 2. 等待全部下游消费者 Ready 后再发请求，
			var out *graph.Stream[[]byte]
			if streaming {
				out = graph.NewStream[[]byte]()
				out.Open()
				out.SetExpectedReady(streamNum)
				ctx.Set(param_stream_out.ID, out)
				defer out.Close()
				if err := out.WaitReady(nRT.GetRtCtx()); err != nil {
					return err
				}
			}

			// 3. send request
			data, err := json.Marshal(body)
			if err != nil {
				return err
			}
			req, err := http.NewRequestWithContext(nRT.GetRtCtx(), http.MethodPost, endpoint, bytes.NewReader(data))
			if err != nil {
				return err
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+api)

			client := &http.Client{} // 无超时：TTS 可能较慢，靠 ctx 中断
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
				return fmt.Errorf("mlx-speech: %s: %s", resp.Status, strings.TrimSpace(string(b)))
			}

			// 4. 分块读取：原始二进制块原样送流（下游直接转发到前端），同时累积全量。
			buf := make([]byte, 64*1024)
			var full []byte
			for {
				n, rerr := resp.Body.Read(buf)
				if n > 0 {
					full = append(full, buf[:n]...)
					if streaming {
						chunk := make([]byte, n)
						copy(chunk, buf[:n])
						out.SendSafe(nRT.GetRtCtx(), chunk)
					}
				}
				if rerr == io.EOF {
					break
				}
				if rerr != nil {
					return rerr
				}
			}

			ctx.Set(param_audio_out.ID, base64.StdEncoding.EncodeToString(full))
			return nil
		},
	})
}
