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

	paramContainer := graph.Parameter{
		ID:      "container_id",
		Kind:    graph.ParamString,
		Default: "",
	}

	paramLog := graph.Parameter{
		ID:      "log",
		Kind:    graph.ParamString,
		Connect: []graph.ParamConnect{graph.ConnOutput},
		Default: "",
	}

	manager.Register(&graph.NodeSpec{
		ID:                 "Exec--ExecDocker",
		RequiredPermission: graph.PermissionDangerous,
		Name:               "Exec Docker",
		Description:        `Executes a full shell command in a Docker container (no whitelist/permission check) \n Run: docker exec <container_id> sh -c <command> (requires the docker CLI on PATH). \n log: combined stdout+stderr (interleaved); trailing [exit code: N] only when non-zero.`,
		Plist: []*graph.Parameter{
			&paramCommand,
			&paramContainer,
			&paramLog,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			// 1. prepare (完整 shell 命令字符串 + 容器 id)。
			command := ctx.Get(paramCommand.ID).(string)
			container := ctx.Get(paramContainer.ID).(string)

			// 2. execute in docker container: docker exec <container> sh -c <command>
			cmd := graph.NewCmd(nRT.GetRtCtx(), "docker", "exec", container, "sh", "-c", command)
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
			logText := logBuf.String()
			if exitCode != 0 {
				logText += fmt.Sprintf("\n[exit code: %d]", exitCode)
			}
			ctx.Set(paramLog.ID, logText)
			return nil
		},
	})
}
