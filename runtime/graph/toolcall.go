package graph

import (
	"encoding/json"
	"fmt"
)

// 能力调用请求，直接用 OpenAI 协议的 tool call 形状，function.arguments 是 JSON 字符串、用 Args() 解析。
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// 解析 OpenAI tool call 并做最小校验：function.name 必填，arguments 若存在必须是合法 JSON 对象字符串。
func ParseToolCall(raw json.RawMessage) (*ToolCall, error) {
	var tc ToolCall
	if err := json.Unmarshal(raw, &tc); err != nil {
		return nil, fmt.Errorf("tool call: failed to parse: %w", err)
	}
	if tc.Function.Name == "" {
		return nil, fmt.Errorf("tool call: missing function.name")
	}
	if tc.Function.Arguments != "" {
		var obj map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &obj); err != nil {
			return nil, fmt.Errorf("tool call: function.arguments is not a valid JSON object string: %w", err)
		}
	}
	return &tc, nil
}

// 把 function.arguments 解析成对象，空串/缺省/解析失败返回空 map。
func (tc *ToolCall) Args() map[string]any {
	args := map[string]any{}
	if tc.Function.Arguments == "" {
		return args
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil || args == nil {
		return map[string]any{}
	}
	return args
}
