package nodes

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/noddock/noddock/runtime/graph"
)

// payload 是 OpenAI /chat/completions 请求体。
type payload struct {
	Model          string            `json:"model"`
	ExtraBody      string            `json:"extra_body,omitempty"`
	Messages       []PayMessage      `json:"messages"`
	ResponseFormat *ResponseFormat   `json:"response_format,omitempty"`
	ToolChoice     any               `json:"tool_choice,omitempty"`
	MaxTokens      int               `json:"max_tokens,omitempty"`
	Temperature    float64           `json:"temperature,omitempty"`
	Stream         bool              `json:"stream,omitempty"`
	Tools          []json.RawMessage `json:"tools,omitempty"`
}

// PayMessage 是 OpenAI 消息格式。
// Content 用 any：兼容纯文本（string）和多模态（[]content part，含 image_url）。
type PayMessage struct {
	Role       string           `json:"role"`
	Content    any              `json:"content"`
	ToolCalls  []graph.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

// ResponseFormat 是 OpenAI 响应格式（json_schema）。
type ResponseFormat struct {
	Type       string          `json:"type"`
	JsonSchema json.RawMessage `json:"json_schema"`
}

// dump 是非流式响应体。
type dump struct {
	Choices []struct {
		Message PayMessage `json:"message"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// streamChunk 是流式响应的 chunk（SSE data 行）。
type streamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Choices []struct {
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
		Delta        struct {
			Role      string           `json:"role,omitempty"`
			Content   string           `json:"content,omitempty"`
			ToolCalls []streamToolCall `json:"tool_calls,omitempty"`
		} `json:"delta"`
	} `json:"choices"`
}

// streamToolCall 是流式响应中的 tool_calls 分片（带 index，用于按序聚合）。
type streamToolCall struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// applySettingsExtra 把 settings_extra（JSON 对象）合并进请求 body：同名键覆盖、新键新增。
// 空对象（{}）原样返回，不做任何改动。
func applySettingsExtra(body []byte, extra json.RawMessage) ([]byte, error) {
	if len(extra) <= 2 {
		return body, nil
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("settings_extra merge failed (body is not a JSON object): %w", err)
	}
	var e map[string]any
	if err := json.Unmarshal(extra, &e); err != nil {
		return nil, fmt.Errorf("settings_extra must be a JSON object: %w", err)
	}
	for k, v := range e {
		m[k] = v
	}
	return json.Marshal(m)
}

// parseSSELine 解析一行 SSE 数据。
// 返回 (chunk, true)：遇到 data: [DONE] 结束标记；
// 返回 (nil, false)：该行应跳过（空行/注释/非 data 行/解析失败）。
func parseSSELine(line string) (*streamChunk, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, ":") {
		return nil, false
	}
	if line == "data: [DONE]" {
		return nil, true
	}
	if !strings.HasPrefix(line, "data: ") {
		return nil, false
	}
	var chunk streamChunk
	if err := json.Unmarshal([]byte(line[6:]), &chunk); err != nil {
		return nil, false
	}
	return &chunk, false
}

// accumulateToolCalls 将流式 tool_calls 分片按 index 聚合。
// 流式协议中 id/type/function.name 仅在首个分片携带，arguments 需逐片拼接。
func accumulateToolCalls(agg []streamToolCall, deltas []streamToolCall) []streamToolCall {
	for _, d := range deltas {
		idx := -1
		for i := range agg {
			if agg[i].Index == d.Index {
				idx = i
				break
			}
		}
		if idx < 0 {
			agg = append(agg, d)
			continue
		}
		if d.ID != "" {
			agg[idx].ID = d.ID
		}
		if d.Type != "" {
			agg[idx].Type = d.Type
		}
		if d.Function.Name != "" {
			agg[idx].Function.Name = d.Function.Name
		}
		agg[idx].Function.Arguments += d.Function.Arguments
	}
	return agg
}

// finalizeToolCalls 将聚合后的流式 tool_calls 按 index 排序，转换为协议要求的完整 ToolCall 列表。
func finalizeToolCalls(agg []streamToolCall) []graph.ToolCall {
	if len(agg) == 0 {
		return nil
	}
	sort.Slice(agg, func(i, j int) bool { return agg[i].Index < agg[j].Index })
	calls := make([]graph.ToolCall, 0, len(agg))
	for _, a := range agg {
		var call graph.ToolCall
		call.ID = a.ID
		call.Type = a.Type
		call.Function.Name = a.Function.Name
		call.Function.Arguments = a.Function.Arguments
		calls = append(calls, call)
	}
	return calls
}
