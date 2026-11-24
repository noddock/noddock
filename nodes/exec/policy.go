package nodes

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

type PermissionLevel string

const (
	PermissionRead  PermissionLevel = "read"
	PermissionWrite PermissionLevel = "write"
)

func permissionValue(p PermissionLevel) int {
	switch p {
	case PermissionRead:
		return 0
	case PermissionWrite:
		return 1
	default:
		return -1
	}
}

type SubCommandPolicy struct {
	// 子命令是否允许
	Allow bool
	// 该子命令需要的权限
	Permission PermissionLevel
}

type CommandPolicy struct {
	Command     string
	Allow       bool
	Permission  PermissionLevel
	SubCommands map[string]SubCommandPolicy
}

// DefaultPolicies 是 exec_safe 节点 whitelist 参数的默认值（常用安全命令）。
//
// 设计：
//   - read 权限：ls/cat/grep/head/tail/wc/find/echo/pwd/date/git status|diff|log
//   - write 权限：mkdir/trash/touch/cp/mv/git add|commit
//   - 禁止：rm/bash/sh/sudo/curl/wget/git push|config
var DefaultPolicies = map[string]CommandPolicy{
	// read 权限（只读命令）。
	"ls": {
		Command:    "ls",
		Allow:      true,
		Permission: PermissionRead,
	},
	"cat": {
		Command:    "cat",
		Allow:      true,
		Permission: PermissionRead,
	},
	"grep": {
		Command:    "grep",
		Allow:      true,
		Permission: PermissionRead,
	},
	"head": {
		Command:    "head",
		Allow:      true,
		Permission: PermissionRead,
	},
	"tail": {
		Command:    "tail",
		Allow:      true,
		Permission: PermissionRead,
	},
	"wc": {
		Command:    "wc",
		Allow:      true,
		Permission: PermissionRead,
	},
	"find": {
		Command:    "find",
		Allow:      true,
		Permission: PermissionRead,
	},
	"echo": {
		Command:    "echo",
		Allow:      true,
		Permission: PermissionRead,
	},
	"pwd": {
		Command:    "pwd",
		Allow:      true,
		Permission: PermissionRead,
	},
	"date": {
		Command:    "date",
		Allow:      true,
		Permission: PermissionRead,
	},
	// write 权限（写命令）。
	"mkdir": {
		Command:    "mkdir",
		Allow:      true,
		Permission: PermissionWrite,
	},
	"trash": {
		Command:    "trash",
		Allow:      true,
		Permission: PermissionWrite,
	},
	"touch": {
		Command:    "touch",
		Allow:      true,
		Permission: PermissionWrite,
	},
	"cp": {
		Command:    "cp",
		Allow:      true,
		Permission: PermissionWrite,
	},
	"mv": {
		Command:    "mv",
		Allow:      true,
		Permission: PermissionWrite,
	},
	// git（子命令白名单）。
	"git": {
		Command:    "git",
		Allow:      true,
		Permission: PermissionRead,
		SubCommands: map[string]SubCommandPolicy{
			"status": {
				Allow:      true,
				Permission: PermissionRead,
			},
			"diff": {
				Allow:      true,
				Permission: PermissionRead,
			},
			"log": {
				Allow:      true,
				Permission: PermissionRead,
			},
			"add": {
				Allow:      true,
				Permission: PermissionWrite,
			},
			"commit": {
				Allow:      true,
				Permission: PermissionWrite,
			},
			"push": {
				Allow: false, // 禁止（网络操作）。
			},
			"config": {
				Allow: false, // 禁止（修改配置）。
			},
		},
	},
	// 禁止的命令。
	"rm": {
		Command: "rm",
		Allow:   false, // 禁止（用 trash 替代）。
	},
	"bash": {
		Command: "bash",
		Allow:   false,
	},
	"sh": {
		Command: "sh",
		Allow:   false,
	},
	"sudo": {
		Command: "sudo",
		Allow:   false,
	},
	"curl": {
		Command: "curl",
		Allow:   false, // 禁止（网络操作）。
	},
	"wget": {
		Command: "wget",
		Allow:   false, // 禁止（网络操作）。
	},
}

// CheckExecPermission 检查命令是否在白名单里，以及权限是否足够。
// policies 是节点参数（whitelist）解析后的白名单。
func CheckExecPermission(policies map[string]CommandPolicy, command string, args []string, requestStr string) error {
	if !slices.Contains([]string{"read", "write"}, requestStr) {
		return errors.New("invalid permission (want read/write)")
	}
	request := PermissionLevel(requestStr)

	policy, ok := policies[command]
	if !ok {
		return fmt.Errorf("command %s is not allowed", command)
	}
	if !policy.Allow {
		return fmt.Errorf("command %s is forbidden", command)
	}
	if permissionValue(request) < permissionValue(policy.Permission) {
		return fmt.Errorf("command %s requires %s permission", command, policy.Permission)
	}
	// 没有子命令
	if len(policy.SubCommands) == 0 {
		return nil
	}
	if len(args) == 0 {
		return fmt.Errorf("command %s requires subcommand", command)
	}
	sub := args[0]
	subPolicy, ok := policy.SubCommands[sub]
	if !ok {
		return fmt.Errorf("subcommand %s %s is not allowed", command, sub)
	}
	if !subPolicy.Allow {
		return fmt.Errorf("%s %s is forbidden", command, sub)
	}
	if permissionValue(request) < permissionValue(subPolicy.Permission) {
		return fmt.Errorf("%s %s requires %s permission", command, sub, subPolicy.Permission)
	}
	return nil
}

// CheckEscape 检查**所有参数**是否逃逸（绝对路径 / .. 父路径）。
// 重新设计：不只检查 PathArgs，而是检查所有参数。这样即使命令没设 PathArgs，
// 绝对/相对路径也能被拦截。
func CheckEscape(args []string) error {
	for _, arg := range args {
		if err := validatePathArg(arg); err != nil {
			return err
		}
	}
	return nil
}

func validatePathArg(arg string) error {
	// 绝对路径禁止。
	if filepath.IsAbs(arg) {
		return errors.New("absolute path is not allowed: " + arg)
	}
	// .. 父路径禁止。
	clean := filepath.Clean(arg)
	for _, p := range strings.Split(clean, string(filepath.Separator)) {
		if p == ".." {
			return errors.New("parent path escape is not allowed: " + arg)
		}
	}
	return nil
}
