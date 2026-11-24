package nodes

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

// listCmd 是查询可用指令的特殊命令值（返回白名单里的命令列表，不执行）。
const listCmd = "__list__"

func init() {
	// 白名单默认值（DefaultPolicies 序列化）。
	defaultWhitelist, _ := json.Marshal(DefaultPolicies)

	paramInput := graph.Parameter{
		ID:      "input",
		Kind:    graph.ParamJSON,
		Connect: []graph.ParamConnect{graph.ConnInput},
		Default: `{"command":"ls", "args":["-al"]}`,
	}

	paramWhitelist := graph.Parameter{
		ID:      "whitelist",
		Kind:    graph.ParamJSON,
		Default: string(defaultWhitelist),
	}

	paramPermission := graph.Parameter{
		ID:         "permission",
		Kind:       graph.ParamString,
		Default:    "read",
		Selections: []string{"read", "write"},
	}

	paramLog := graph.Parameter{
		ID:      "log",
		Kind:    graph.ParamString,
		Connect: []graph.ParamConnect{graph.ConnOutput},
		Default: "",
	}

	manager.Register(&graph.NodeSpec{
		ID:                 "Exec--ExecSafe",
		RequiredPermission: graph.PermissionDangerous,
		Name:               "Exec Safe",
		Description:        `Executes a shell command locally with whitelist + permission + escape prevention \n input: {"command":"ls","args":["-la"]} \n log: combined stdout+stderr (interleaved); trailing [exit code: N] only when non-zero. \n Note: the input is command+args (NOT a full command, to prevent pipe bypass); the whitelist parameter defaults to the built-in safe commands (check in Whitelist ); no absolute path, no .. (checks all args); a policy violation puts the error in log with exit_code=126 and does not abort the run; use command='__list__' to list available commands. \n If you wish to edit your own whitelist, be careful when adding commands like python or npx which allow llm writes and runs script to escape the white list.`,
		Plist: []*graph.Parameter{
			&paramInput,
			&paramWhitelist,
			&paramPermission,
			&paramLog,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			// 1. prepare
			var input ExecIn
			raw := ctx.Get(paramInput.ID).(json.RawMessage)
			if err := json.Unmarshal(raw, &input); err != nil {
				return err
			}
			command := input.Command
			args := input.Args
			permission := ctx.Get(paramPermission.ID).(string)
			whitelistRaw := ctx.Get(paramWhitelist.ID).(json.RawMessage)

			// 2. parse whitelist (node param). 白名单解析失败是配置错误（return err）。
			var policies map[string]CommandPolicy
			if err := json.Unmarshal(whitelistRaw, &policies); err != nil {
				return err
			}

			// 3. 特殊命令：__list__ → 返回白名单里可用的指令（不执行）。
			if command == listCmd {
				keys := make([]string, 0, len(policies))
				for k := range policies {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				logText := "可用指令 (available commands):\n" + strings.Join(keys, "\n")
				ctx.Set(paramLog.ID, logText)
				return nil
			}

			// 4. check permission (whitelist). 触发 → 错误进 log + exit_code=126（不是 run 抛错）。
			if err := CheckExecPermission(policies, command, args, permission); err != nil {
				logText := err.Error() + "\n使用 command='__list__' 查询可用指令" +
					fmt.Sprintf("\n[exit code: 126]")
				ctx.Set(paramLog.ID, logText)
				return nil
			}

			// 5. check escape (所有参数). 触发 → 错误进 log + exit_code=126（不是 run 抛错）。
			if err := CheckEscape(args); err != nil {
				logText := err.Error() + fmt.Sprintf("\n[exit code: 126]")
				ctx.Set(paramLog.ID, logText)
				return nil
			}

			// 6. execute (local).
			cmd := graph.NewCmd(nRT.GetRtCtx(), command, args...)
			cmd.Dir = nRT.GetProjectDir()
			var logBuf lockedBuffer
			cmd.Stdout = &logBuf
			cmd.Stderr = &logBuf // 同一 buffer → stdout/stderr 按到达顺序交错
			err := cmd.Run()
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					return err
				}
			}
			logText := string(graph.FixEncoding([]byte(logBuf.String())))
			if exitCode != 0 {
				logText += fmt.Sprintf("\n[exit code: %d]", exitCode)
			}
			ctx.Set(paramLog.ID, logText)
			return nil
		},
	})
}
