package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
	"github.com/noddock/noddock/runtime/tool"
	"github.com/noddock/noddock/runtime/tool/fo"
)

const contextTable = "context_message"

func init() {
	paramDbAddr := graph.Parameter{
		ID:      "db_addr",
		Name:    "db addr",
		Kind:    graph.ParamString,
		Default: "context",
	}
	paramWrite := graph.Parameter{
		ID:      "write",
		Kind:    graph.ParamJSON,
		Connect: []graph.ParamConnect{graph.ConnInput},
		Default: "[]",
	}
	paramRead := graph.Parameter{
		ID:      "read",
		Kind:    graph.ParamJSON,
		Connect: []graph.ParamConnect{graph.ConnOutput},
		Default: "[]",
	}

	manager.Register(&graph.NodeSpec{
		ID:          "DB--Context",
		Name:        "DB Context",
		Description: `LLM context storage. Write if "write" input is connected (OpenAI []message, role auto-detects user/system/assistant/tool); read (OpenAI messages, time-ordered) if "read" output is connected.`,
		Plist: []*graph.Parameter{
			&paramDbAddr,
			&paramWrite,
			&paramRead,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			dbAddr := ctx.Get(paramDbAddr.ID).(string)
			if !strings.HasSuffix(dbAddr, ".sqlite") {
				dbAddr += ".sqlite"
			}
			abs, err := fo.GetPath(dbAddr, nRT.GetPID(), false, true)
			if err != nil {
				return fmt.Errorf("DB--Context: %w", err)
			}

			id := "db:" + dbAddr
			var conn *dbConn
			if nRT.RecourceHas(id) {
				if v, err := nRT.RecourceGet(id); err == nil {
					conn, _ = v.(*dbConn)
				}
			}
			if conn == nil {
				var err error
				conn, err = openDB("sqlite", abs)
				if err != nil {
					return fmt.Errorf("DB--Context: failed to open database: %w", err)
				}
				conn.id = id
				nRT.RecourceSet(conn)
			}

			if err := ensureContextTable(nRT.GetRtCtx(), conn); err != nil {
				return fmt.Errorf("DB--Context: failed to create table: %w", err)
			}

			now := time.Now().UTC().Format(time.RFC3339)

			// write 有上游连接 → 写入；read 有下游连接 → 读取（非互斥，先 write 后 read）。
			if nRT.UpstreamCount(paramWrite.ID) > 0 {
				input := ctx.Get(paramWrite.ID).(json.RawMessage)
				if err := writeMessages(nRT, conn, input, now); err != nil {
					return err
				}
			}
			if nRT.DownstreamCount(paramRead.ID) > 0 {
				if err := readMessages(ctx, nRT, conn); err != nil {
					return err
				}
			}
			return nil
		},
	})
}

// writeMessages 把 OpenAI []message 逐条转成 db 格式保存。
func writeMessages(nRT graph.NodeRunTool, conn *dbConn, input json.RawMessage, now string) error {
	var messages []struct {
		Role       string           `json:"role"`
		Content    json.RawMessage  `json:"content"`
		ToolCalls  []graph.ToolCall `json:"tool_calls"`
		ToolCallID string           `json:"tool_call_id"`
	}
	if err := json.Unmarshal(input, &messages); err != nil {
		return fmt.Errorf("DB--Context: write input must be an OpenAI []message array: %w", err)
	}

	for _, msg := range messages {
		if msg.Role == "tool" {
			// tool result：UPDATE 对应行（message_id=tool_call_id）的 response。
			var contentStr string
			_ = json.Unmarshal(msg.Content, &contentStr) // JSON string→纯字符串（null→""）
			if _, err := conn.exec(nRT.GetRtCtx(),
				fmt.Sprintf("UPDATE %s SET response_content = ?, response_created = ? WHERE message_id = ?", contextTable),
				[]any{contentStr, now, msg.ToolCallID}); err != nil {
				return err
			}
			continue
		}

		// assistant 带 tool_calls：每个 tool call 独立存一行（content 丢弃，通常只是描述无意义）。
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				tcJSON, err := json.Marshal(tc)
				if err != nil {
					return err
				}
				if _, err := conn.exec(nRT.GetRtCtx(),
					fmt.Sprintf("INSERT INTO %s (message_id, request_owner, request_content, request_created) VALUES (?, ?, ?, ?)", contextTable),
					[]any{tc.ID, "assistant", string(tcJSON), now}); err != nil {
					return err
				}
			}
			continue
		}

		// user/system/assistant（无 tool_calls）：INSERT 一行（request）。
		messageID := fmt.Sprintf("msg_%d", time.Now().UnixNano())
		var contentStr string
		_ = json.Unmarshal(msg.Content, &contentStr) // JSON string→纯字符串（null→""）
		if _, err := conn.exec(nRT.GetRtCtx(),
			fmt.Sprintf("INSERT INTO %s (message_id, request_owner, request_content, request_created) VALUES (?, ?, ?, ?)", contextTable),
			[]any{messageID, msg.Role, contentStr, now}); err != nil {
			return err
		}
	}
	return nil
}

// readMessages 读取全量（按时间排序），转成 OpenAI 格式 messages 输出到 read 参数。
func readMessages(ctx *graph.Context, nRT graph.NodeRunTool, conn *dbConn) error {
	out, err := conn.query(nRT.GetRtCtx(),
		fmt.Sprintf("SELECT message_id, request_owner, request_content, response_content, request_created, response_created FROM %s", contextTable),
		nil)
	if err != nil {
		return err
	}

	type event struct {
		timestamp string
		message   map[string]any
	}
	events := make([]event, 0)
	for _, row := range out["rows"].([]map[string]any) {
		messageID := ctxToStr(row["message_id"])
		requestOwner := ctxToStr(row["request_owner"])
		requestContent := ctxToStr(row["request_content"])
		responseContent := ctxToStr(row["response_content"])
		requestCreated := ctxToStr(row["request_created"])
		responseCreated := ctxToStr(row["response_created"])

		if requestContent != "" {
			events = append(events, event{requestCreated, requestToOpenAI(requestOwner, requestContent)})
		}
		if responseContent != "" {
			events = append(events, event{responseCreated, responseToOpenAI(messageID, responseContent)})
		}
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].timestamp < events[j].timestamp })

	messages := make([]map[string]any, 0, len(events))
	for _, e := range events {
		messages = append(messages, e.message)
	}
	ctx.Set("read", tool.Any2JsonRaw(messages))
	return nil
}

// requestToOpenAI 把 request 转成 OpenAI 消息。
// tool call（request_content 可解析为 ToolCall 且有 id）→ assistant + tool_calls（单条）；
// 否则按 owner 前缀定 role，content 原样。
func requestToOpenAI(requestOwner, requestContent string) map[string]any {
	var tc graph.ToolCall
	if err := json.Unmarshal([]byte(requestContent), &tc); err == nil && tc.ID != "" {
		return map[string]any{"role": "assistant", "tool_calls": []any{tc}}
	}
	role := "assistant"
	if strings.HasPrefix(requestOwner, "user") {
		role = "user"
	} else if strings.HasPrefix(requestOwner, "system") {
		role = "system"
	}
	return map[string]any{"role": role, "content": requestContent}
}

// responseToOpenAI 把 response 转成 OpenAI tool 消息。
func responseToOpenAI(messageID, responseContent string) map[string]any {
	return map[string]any{"role": "tool", "tool_call_id": messageID, "content": responseContent}
}

// ctxToStr 把 any 转成 string（nil→""）。
func ctxToStr(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// ensureContextTable 建表（如果不存在）。以 message 为行单位。
func ensureContextTable(ctx context.Context, conn *dbConn) error {
	_, err := conn.exec(ctx, `
CREATE TABLE IF NOT EXISTS `+contextTable+` (
	id 					INTEGER PRIMARY KEY AUTOINCREMENT,
	message_id 			TEXT NOT NULL,
	request_owner 		TEXT NOT NULL,
	request_content 	TEXT NOT NULL DEFAULT '',
	response_content 	TEXT NOT NULL DEFAULT '',
	request_created 	TEXT NOT NULL,
	response_created 	TEXT NOT NULL DEFAULT ''
);`, nil)
	return err
}
