package nodes

import (
	"fmt"
	"os/exec"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {
	paramCommand := graph.Parameter{
		ID:      "command",
		Kind:    graph.ParamString,
		Connect: []graph.ParamConnect{graph.ConnInput},
		Default: "",
	}

	paramLog := graph.Parameter{
		ID:      "log",
		Kind:    graph.ParamString,
		Connect: []graph.ParamConnect{graph.ConnOutput},
		Default: "",
	}

	manager.Register(&graph.NodeSpec{
		ID:                 "Exec--Exec",
		RequiredPermission: graph.PermissionDangerous,
		Name:               "Exec",
		Description:        `Executes a full shell command locally (no whitelist/permission check) \n Run: sh -c <command> (unix) / cmd.exe /c <command> (windows) \n log: combined stdout+stderr (interleaved); trailing [exit code: N] only when non-zero. \n working dir = project workspace \n non-zero exit codes are not run errors; the command is interruptible.`,
		Plist: []*graph.Parameter{
			&paramCommand,
			&paramLog,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			// 1. prepare (完整 shell 命令字符串)。
			command := ctx.Get(paramCommand.ID).(string)

			// 2. execute (跨平台 shell：unix sh -c / windows cmd.exe /c + 平台进程属性，见 core/graph/shell.go)。
			cmd := graph.NewShellCmd(nRT.GetRtCtx(), command, nRT.GetProjectDir())
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
