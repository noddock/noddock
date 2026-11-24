package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
	"github.com/noddock/noddock/runtime/tool"
)

type mcpServerConfig struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Timeout int               `json:"timeout,omitempty"`
}

type mcpConfig struct {
	MCPServers map[string]mcpServerConfig `json:"mcpServers"`
}

type mcpConn struct {
	Session *mcp.ClientSession
	Timeout int
	mu      sync.Mutex
}

func (c *mcpConn) Close() error {
	if c == nil || c.Session == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Session.Close()
}

// server name → connection
type mcpResource struct {
	Conns map[string]*mcpConn
	id    string // 资源标识（= 资源表 key）
}

func (r *mcpResource) ID() string { return r.id }

func (r *mcpResource) Close() error {
	if r == nil {
		return nil
	}
	var firstErr error
	for _, conn := range r.Conns {
		if err := conn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

const mcpDefaultTimeout = 30000 // 毫秒

func init() {
	paramConfig := graph.Parameter{
		ID:      "config",
		Kind:    graph.ParamJSON,
		Default: `{"mcpServers":{"<server>":{"command":"<cmd>","args":[],"url":"","headers":{}}}}`,
		Connect: []graph.ParamConnect{graph.ConnEdit, graph.ConnInput},
	}
	paramCall := graph.Parameter{
		ID:      "call",
		Name:    "tool call",
		Kind:    graph.ParamJSON,
		Default: `{"id":"","type":"function","function":{"name":"mcp__<server>__<tool>","arguments":"{}"}}`,
		Connect: []graph.ParamConnect{graph.ConnInput},
	}
	paramTools := graph.Parameter{
		ID:      "tools",
		Name:    "tool list",
		Kind:    graph.ParamJSON,
		Default: "[]",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}
	paramResult := graph.Parameter{
		ID:      "result",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:                 "MCP--McpClient",
		RequiredPermission: graph.PermissionDangerous,
		Name:               "MCP Client",
		Description:        `An external MCP client (lazy loading). Empty call = initialize only; non-empty call = lazy-init if needed, then execute.`,
		Plist: []*graph.Parameter{
			&paramConfig,
			&paramCall,
			&paramTools,
			&paramResult,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			niid := nRT.GetNIID()
			id := "mcp:" + niid

			// 1. 取 resource（基于 NIID），不存在则初始化（懒加载）
			var res *mcpResource
			if nRT.RecourceHas(id) {
				if v, err := nRT.RecourceGet(id); err == nil {
					res, _ = v.(*mcpResource)
				}
			}
			if res == nil {
				var err error
				res, err = initMcpServers(ctx, nRT)
				if err != nil {
					return err
				}
				res.id = id
				nRT.RecourceSet(res)
			}

			// 2. 解析 call（放在取 resource 后面）
			callRaw, _ := ctx.Get("call").(json.RawMessage)
			call, err := graph.ParseToolCall(callRaw)
			if err != nil {
				ctx.Set("result", tool.Any2JsonRaw(map[string]any{"status": "error", "code": "MCP_INIT_ONLY", "error": "call is empty or invalid: " + err.Error()}))
				return nil
			}

			// 3. 处理 call：拆 server/tool，取连接，执行 tools/call
			name := call.Function.Name
			if !strings.HasPrefix(name, "mcp__") {
				ctx.Set("result", tool.Any2JsonRaw(map[string]any{"status": "error", "code": "MCP_CALL", "error": "call name 需以 mcp__ 开头", "call_id": call.ID}))
				return nil
			}
			rest := strings.TrimPrefix(name, "mcp__")
			i := strings.Index(rest, "__")
			if i <= 0 || i+2 >= len(rest) {
				ctx.Set("result", tool.Any2JsonRaw(map[string]any{"status": "error", "code": "MCP_CALL", "error": "call name 需为 mcp__<server>__<tool>", "call_id": call.ID}))
				return nil
			}
			server := rest[:i]
			toolCall := rest[i+2:]

			conn, ok := res.Conns[server]
			if !ok || conn.Session == nil {
				ctx.Set("result", tool.Any2JsonRaw(map[string]any{"status": "error", "code": "MCP_NO_CONN", "error": fmt.Sprintf("server %q not connected (check config / init status)", server), "call_id": call.ID}))
				return nil
			}

			// 执行 tools/call（基于 rtCtx，可被 interrupt 取消）。
			conn.mu.Lock()
			callCtx, callCancel := context.WithTimeout(nRT.GetRtCtx(), time.Duration(conn.Timeout)*time.Millisecond)
			res2, cerr := conn.Session.CallTool(callCtx, &mcp.CallToolParams{
				Name:      toolCall,
				Arguments: call.Args(),
			})
			callCancel()
			conn.mu.Unlock()
			if cerr != nil {
				ctx.Set("result", tool.Any2JsonRaw(map[string]any{"status": "error", "code": "MCP_CALL_FAILED", "error": cerr.Error(), "server": server, "tool": toolCall, "call_id": call.ID}))
				return nil
			}

			out := map[string]any{"server": server, "tool": toolCall, "call_id": call.ID}
			if raw := tool.Any2JsonRaw(res2); len(raw) > 0 {
				_ = json.Unmarshal(raw, &out) // content / structuredContent / isError 原样带出
			}
			switch {
			case res2.NeedsInput():
				out["status"] = "error"
				out["code"] = "MCP_INPUT_REQUIRED"
			case res2.IsError:
				out["status"] = "error"
				out["code"] = "MCP_TOOL_ERROR"
			default:
				out["status"] = "ok"
			}
			ctx.Set("result", tool.Any2JsonRaw(out))
			return nil
		},
	})
}

// initMcpServers 读 config，为每个 server 建连接 + 拉 tools，返回 mcpResource（内部 map 存连接）。同时更新 tools 出口（OpenAI 格式，可直接接 LLM）。部分 server 失败不中断（记录在 result，由调用方处理）。
func initMcpServers(ctx *graph.Context, nRT graph.NodeRunTool) (*mcpResource, error) {
	cfgRaw := ctx.Get("config").(json.RawMessage)
	var cfg mcpConfig

	if err := json.Unmarshal(cfgRaw, &cfg); err != nil {
		return nil, fmt.Errorf("Mcp config must be in mcp.json form: %w", err)
	}
	if len(cfg.MCPServers) == 0 {
		return nil, fmt.Errorf("Mcp config has no mcpServers")
	}

	res := &mcpResource{Conns: map[string]*mcpConn{}}
	tools := []map[string]any{}
	statuses := []map[string]any{}
	allOK := true
	for name, sc := range cfg.MCPServers {
		timeoutMs := sc.Timeout
		if timeoutMs <= 0 {
			timeoutMs = mcpDefaultTimeout
		}
		var transport mcp.Transport
		switch {
		case sc.Command != "":
			cmd := graph.NewCmd(context.Background(), sc.Command, sc.Args...)
			envs := os.Environ()
			for k, v := range sc.Env {
				envs = append(envs, k+"="+v)
			}
			cmd.Env = envs
			transport = &mcp.CommandTransport{Command: cmd}
		case sc.URL != "":
			transport = &mcp.StreamableClientTransport{
				Endpoint:             sc.URL,
				HTTPClient:           &http.Client{Transport: mcpHeaderTransport{headers: sc.Headers}},
				DisableStandaloneSSE: true,
			}
		default:
			return nil, fmt.Errorf("Mcp--Mcp Client: server %q needs command or url", name)
		}

		client := mcp.NewClient(&mcp.Implementation{Name: "noddock", Version: "1.0.0"}, nil)
		connectCtx, cancel := context.WithTimeout(nRT.GetRtCtx(), time.Duration(timeoutMs)*time.Millisecond)
		session, cerr := client.Connect(connectCtx, transport, nil)
		cancel()
		if cerr != nil {
			statuses = append(statuses, map[string]any{"server": name, "ok": false, "error": cerr.Error()})
			allOK = false
			continue
		}

		res.Conns[name] = &mcpConn{Session: session, Timeout: timeoutMs}

		// 拉 tools/list
		listCtx, listCancel := context.WithTimeout(nRT.GetRtCtx(), time.Duration(timeoutMs)*time.Millisecond)
		var list []*mcp.Tool
		var lerr error
		cursor := ""
		for {
			lres, err := session.ListTools(listCtx, &mcp.ListToolsParams{Cursor: cursor})
			if err != nil {
				lerr = err
				break
			}
			list = append(list, lres.Tools...)
			if lres.NextCursor == "" {
				break
			}
			cursor = lres.NextCursor
		}
		listCancel()
		if lerr != nil {
			statuses = append(statuses, map[string]any{"server": name, "ok": false, "error": lerr.Error()})
			allOK = false
			continue
		}
		for _, t := range list {
			// OpenAI 请求 tools 格式
			tools = append(tools, map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        "mcp__" + name + "__" + t.Name,
					"description": t.Description,
					"parameters":  t.InputSchema,
				},
			})
		}
		statuses = append(statuses, map[string]any{"server": name, "ok": true, "tools": len(list)})
	}

	// 更新 tools 出口。
	ctx.Set("tools", tool.Any2JsonRaw(tools))

	out := map[string]any{"status": "ok", "servers": statuses, "tools": len(tools)}
	if !allOK {
		out["status"] = "error"
		out["code"] = "MCP_LOAD_FAILED"
	}
	ctx.Set("result", tool.Any2JsonRaw(out))

	return res, nil
}

type mcpHeaderTransport struct {
	headers map[string]string
}

func (t mcpHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}
	return http.DefaultTransport.RoundTrip(req)
}
