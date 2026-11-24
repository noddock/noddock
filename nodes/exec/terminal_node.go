package nodes

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

// stderrTail 取 stderr 末尾（错误信息通常在最后），截断到 500 字符。
func stderrTail(s string) string {
	const max = 500
	if len(s) > max {
		return "..." + s[len(s)-max:]
	}
	if s == "" {
		return "(空)"
	}
	return s
}

func init() {
	param_command := graph.Parameter{
		ID:      "command",
		Kind:    graph.ParamString,
		Connect: []graph.ParamConnect{graph.ConnEdit},
		Default: "python server.py",
	}
	param_addr := graph.Parameter{
		ID:      "address",
		Kind:    graph.ParamString,
		Connect: []graph.ParamConnect{graph.ConnEdit, graph.ConnOutput},
		Default: "",
	}
	manager.Register(&graph.NodeSpec{
		ID:                 "Exec--Terminal",
		RequiredPermission: graph.PermissionDangerous,
		Name:               "Terminal [WIP]",
		Description:        `Runs a persistent terminal server for long-running commands (>500 ms). output key connects to Python/JS terminal nodes.`,
		Plist: []*graph.Parameter{
			&param_command,
			&param_addr,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			command := ctx.Get(param_command.ID).(string)
			if command == "" {
				return errors.New("Terminal: command is not set")
			}

			addr, _ := ctx.Get(param_addr.ID).(string)
			if addr == "" {
				addr = uuid.NewString()
			}
			id := "terminal:" + addr

			// id 已存在 → 销毁旧进程再启动
			if nRT.RecourceHas(id) {
				rec, _ := nRT.RecourceGet(id)
				if old, ok := rec.(*graph.Terminal); ok {
					old.Close()
				}
				nRT.RecourceDelete(id)
			}

			t, err := graph.NewTerminal(nRT.GetProjectDir(), command, id)
			if err != nil {
				return fmt.Errorf("Terminal: failed to start process: %w", err)
			}

			// 启动检查：常驻进程若在宽限期内退出，视为启动失败（路径错/解释器不存在/脚本语法错等）。
			// 否则 cmd.Start 成功就静默通过，要等下游写 stdin 才发现进程已死。
			select {
			case <-t.Done():
				t.Close()
				return fmt.Errorf("Terminal: process exited immediately after start (stderr: %s)", stderrTail(t.Stderr.String()))
			case <-time.After(500 * time.Millisecond):
			case <-nRT.GetRtCtx().Done():
				t.Close()
				return nRT.GetRtCtx().Err()
			}

			nRT.RecourceSet(t)
			ctx.Set(param_addr.ID, addr)
			return nil
		},
	})
}
