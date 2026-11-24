package tool

import (
	"encoding/json"
	"fmt"
)

// Any2JsonRaw 把任意值序列化成 json.RawMessage，序列化失败 panic（视为编程错误）。
func Any2JsonRaw(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("Any2JsonRaw: marshal failed: %v", err))
	}
	return json.RawMessage(b)
}
