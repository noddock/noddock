package nodes

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {
	// Input
	param_messages := graph.Parameter{
		ID:      "messages",
		Kind:    graph.ParamJSON,
		Default: "[]",
		Connect: []graph.ParamConnect{graph.ConnInput, graph.ConnOutput},
	}
	param_tools := graph.Parameter{
		ID:      "tools",
		Kind:    graph.ParamJSON,
		Default: "[]",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}

	param_endpoint := graph.Parameter{
		ID:      "endpoint",
		Kind:    graph.ParamString,
		Default: "http://127.0.0.1:1234/v1/chat/completions",
	}
	param_api := graph.Parameter{
		ID:      "api",
		Kind:    graph.ParamString,
		Default: "",
	}
	param_model := graph.Parameter{
		ID:      "model",
		Kind:    graph.ParamString,
		Default: "",
	}
	param_settings_extra := graph.Parameter{
		ID:      "settings_extra",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnHidden},
	}
	// Output
	param_reply := graph.Parameter{
		ID:      "reply",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}
	param_tokens := graph.Parameter{
		ID:      "tokens",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnHidden},
	}
	param_stream_out := graph.Parameter{
		ID:      "stream_out",
		Kind:    graph.ParamStream,
		Default: graph.NewStream[string](),
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "LLM--OpenAI--Chat",
		Name:        "Chat Complete",
		Description: `LLM node: messages+tools in, reply/messages/tokens/stream_out out. Use LLM--OpenAI--ToolCallParser to parse tool calls from reply.`,
		Plist: []*graph.Parameter{
			&param_messages,
			&param_tools,
			&param_endpoint,
			&param_api,
			&param_model,
			&param_settings_extra,
			&param_reply,
			&param_tokens,
			&param_stream_out,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			// 1. 读取输入
			messagesJson, _ := ctx.Get(param_messages.ID).(json.RawMessage)
			toolsJson, _ := ctx.Get(param_tools.ID).(json.RawMessage)
			extraJson, _ := ctx.Get(param_settings_extra.ID).(json.RawMessage)
			model, _ := ctx.Get(param_model.ID).(string)
			endpoint, _ := ctx.Get(param_endpoint.ID).(string)
			api, _ := ctx.Get(param_api.ID).(string)
			if api != "" {
				if resolved, err := nRT.VaultLookup(api); err == nil {
					api = resolved
				}
			}
			streamNum := nRT.DownstreamCount(param_stream_out.ID)
			streaming := streamNum > 0

			var messages []PayMessage
			if err := json.Unmarshal(messagesJson, &messages); err != nil {
				return fmt.Errorf("LLM OpenAI: prompt must be a messages array: %w", err)
			}
			// 深拷贝一份用于请求：解析媒体 URL（
			var requestMessages []PayMessage
			if b, err := json.Marshal(messages); err == nil {
				_ = json.Unmarshal(b, &requestMessages)
			} else {
				requestMessages = messages
			}
			resolveMessagesMedia(nRT, requestMessages)
			var tools []json.RawMessage
			if len(toolsJson) > 2 {
				if err := json.Unmarshal(toolsJson, &tools); err != nil {
					return fmt.Errorf("LLM OpenAI: tools must be an array: %w", err)
				}
			}

			// 2. 流式输出通道
			var out *graph.Stream[string]
			var outClosed bool
			closeOut := func() {
				if out != nil && !outClosed {
					outClosed = true
					out.Close()
				}
			}
			if streaming {
				out = graph.NewStream[string]()
				out.Open()
				out.SetExpectedReady(streamNum)
				ctx.Set(param_stream_out.ID, out)
				defer closeOut()
				if err := out.WaitReady(nRT.GetRtCtx()); err != nil {
					return err
				}
			}

			// 3. 构建请求（用解析后的 requestMessages，媒体已是 base64）
			reqBody := payload{
				Model:    model,
				Messages: requestMessages,
				Stream:   streaming,
				Tools:    tools,
			}
			data, err := json.Marshal(reqBody)
			if err != nil {
				return fmt.Errorf("LLM OpenAI: failed to marshal request: %w", err)
			}
			data, err = applySettingsExtra(data, extraJson)
			if err != nil {
				return fmt.Errorf("LLM OpenAI: %w", err)
			}

			// 4. 发送请求
			req, err := http.NewRequestWithContext(nRT.GetRtCtx(), "POST", endpoint, bytes.NewReader(data))
			if err != nil {
				return fmt.Errorf("LLM OpenAI: failed to build request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+api)
			if streaming {
				req.Header.Set("Accept", "text/event-stream")
				req.Header.Set("Cache-Control", "no-cache")
			}

			client := &http.Client{Timeout: 0}
			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("LLM OpenAI: request failed: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("LLM OpenAI: server returned %d: %s", resp.StatusCode, string(body))
			}

			// 5. 解析响应
			var reply PayMessage
			var tokensJson json.RawMessage

			if !streaming {
				// ============ 非流式解析 ============
				var result dump
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					return fmt.Errorf("LLM OpenAI: failed to parse response: %w", err)
				}
				if len(result.Choices) == 0 {
					return fmt.Errorf("LLM OpenAI: no choices returned")
				}
				assistantMsg := result.Choices[0].Message
				reply = PayMessage{
					Role:      "assistant",
					Content:   assistantMsg.Content,
					ToolCalls: assistantMsg.ToolCalls,
				}
				// tokens
				if result.Usage != nil {
					tokensJson, _ = json.Marshal(map[string]int{
						"prompt_tokens":     result.Usage.PromptTokens,
						"completion_tokens": result.Usage.CompletionTokens,
						"total_tokens":      result.Usage.TotalTokens,
					})
				}
			} else {
				// ============ 流式解析 ============
				scanner := bufio.NewScanner(resp.Body)
				buf := make([]byte, 0, 64*1024)
				scanner.Buffer(buf, 1024*1024)

				var contentBuf strings.Builder
				var agg []streamToolCall
				streamFinished := false

				for scanner.Scan() {
					select {
					case <-nRT.GetRtCtx().Done():
						return nRT.GetRtCtx().Err()
					default:
					}

					chunk, done := parseSSELine(scanner.Text())
					if done {
						streamFinished = true
						break
					}
					if chunk == nil || len(chunk.Choices) == 0 {
						continue
					}
					choice := chunk.Choices[0]
					if choice.Delta.Content != "" {
						contentBuf.WriteString(choice.Delta.Content)
						out.SendSafe(nRT.GetRtCtx(), choice.Delta.Content)
					}
					if len(choice.Delta.ToolCalls) > 0 {
						agg = accumulateToolCalls(agg, choice.Delta.ToolCalls)
						for _, tc := range choice.Delta.ToolCalls {
							if tc.ID != "" {
								out.SendSafe(nRT.GetRtCtx(), "run tool "+tc.ID)
							}
						}
					}
					if choice.FinishReason == "tool_calls" || choice.FinishReason == "stop" {
						streamFinished = true
						break
					}
				}
				if err := scanner.Err(); err != nil {
					return err
				}
				if !streamFinished {
					return fmt.Errorf("LLM OpenAI: stream ended unexpectedly (no finish_reason)")
				}

				toolCalls := finalizeToolCalls(agg)
				reply = PayMessage{
					Role:      "assistant",
					Content:   contentBuf.String(),
					ToolCalls: toolCalls,
				}
			}

			// 6. 输出
			messages = append(messages, reply)
			mJson, err := json.Marshal(reply)
			if err != nil {
				return err
			}
			ctx.Set(param_reply.ID, json.RawMessage(mJson))

			msJson, err := json.Marshal(messages)
			if err != nil {
				return err
			}
			ctx.Set(param_messages.ID, json.RawMessage(msJson))

			if tokensJson != nil {
				ctx.Set(param_tokens.ID, tokensJson)
			} else {
				ctx.Set(param_tokens.ID, json.RawMessage(`{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}`))
			}
			return nil
		},
	})
}
