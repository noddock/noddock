// Package eventbus 事件总线的极简 mock：只保留节点编译/运行所需的最小定义，无前端/SSE 传输。
package eventbus

// EventType 事件类型。
type EventType string

const (
	EventSSEClose      EventType = "sse_close"
	EventSseBuildAudio EventType = "sse_build_audio"
	EventLLMChunk      EventType = "llm_chunk"
)

type Event struct {
	Type    EventType      `json:"type"`
	Payload map[string]any `json:"payload"`
	Delta   string         `json:"delta"` // for LLM
}

func Emit(key string, ev Event) {}
