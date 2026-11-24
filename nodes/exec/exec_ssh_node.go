package nodes

import (
	"fmt"
	"net"
	"time"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
	"golang.org/x/crypto/ssh"
)

// sshConn 是资源表里存的活 ssh 连接（持久连接，跨 run 存活）。
type sshConn struct {
	client *ssh.Client
	id     string
}

func (c *sshConn) ID() string { return c.id }

func (c *sshConn) Close() error {
	return c.client.Close()
}

// sshConnID 由 host:port:user 算资源标识（节点查复用 + 构造时共用，保证一致）。
func sshConnID(host, port, user string) string {
	return fmt.Sprintf("ssh:%s:%s:%s", host, port, user)
}

// newSSHConn 建立 ssh 连接（user@host:port，password 认证，15s 拨号超时）。
func newSSHConn(host, port, user, password string) (*sshConn, error) {
	if port == "" {
		port = "22"
	}
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 软件内功能，跳过 host key 验证（之后可加 known_hosts）。
	}
	d, err := net.DialTimeout("tcp", host+":"+port, 15*time.Second)
	if err != nil {
		return nil, err
	}
	cc, _, _, err := ssh.NewClientConn(d, host+":"+port, config)
	if err != nil {
		_ = d.Close()
		return nil, err
	}
	return &sshConn{client: ssh.NewClient(cc, nil, nil), id: sshConnID(host, port, user)}, nil
}

// alive 检查连接是否还活着（尝试开 session）。
func (c *sshConn) alive() bool {
	session, err := c.client.NewSession()
	if err != nil {
		return false
	}
	session.Close()
	return true
}

func init() {
	paramCommand := graph.Parameter{
		ID:      "command",
		Kind:    graph.ParamString,
		Connect: []graph.ParamConnect{graph.ConnInput},
		Default: "",
	}

	paramHost := graph.Parameter{
		ID:      "host",
		Kind:    graph.ParamString,
		Default: "",
	}

	paramUser := graph.Parameter{
		ID:      "user",
		Kind:    graph.ParamString,
		Default: "",
	}

	paramPassword := graph.Parameter{
		ID:      "password",
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
		ID:                 "Exec--ExecSSH",
		RequiredPermission: graph.PermissionDangerous,
		Name:               "Exec SSH",
		Description:        `Executes a full shell command on a remote SSH host (no whitelist/permission check) \n host accepts host:port (port defaults to 22); password supports a vault ref.`,
		Plist: []*graph.Parameter{
			&paramCommand,
			&paramHost,
			&paramUser,
			&paramPassword,
			&paramLog,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			// 1. prepare (完整 shell 命令字符串 + host:port + user + password)。
			command := ctx.Get(paramCommand.ID).(string)
			host := ctx.Get(paramHost.ID).(string)
			user := ctx.Get(paramUser.ID).(string)
			password := ctx.Get(paramPassword.ID).(string)
			// password 支持 vault ref（解析失败回退原值）。
			if password != "" {
				if resolved, err := nRT.VaultLookup(password); err == nil {
					password = resolved
				}
			}
			// host 接受 host:port（port 缺省 22）。
			port := "22"
			if h, p, err := net.SplitHostPort(host); err == nil {
				host, port = h, p
			}

			// 2. get or create the ssh connection (stored in resource, cross-run; key=ID).
			id := sshConnID(host, port, user)
			var conn *sshConn
			if nRT.RecourceHas(id) {
				if v, err := nRT.RecourceGet(id); err == nil {
					conn, _ = v.(*sshConn)
				}
			}
			// 如果连接断了（或不存在），重新连接。
			if conn == nil || !conn.alive() {
				if conn != nil {
					_ = conn.Close()
					_ = nRT.RecourceDelete(id)
				}
				var err error
				conn, err = newSSHConn(host, port, user, password)
				if err != nil {
					return fmt.Errorf("Exec--ExecSSH: failed to connect ssh: %w", err)
				}
				nRT.RecourceSet(conn)
			}

			// 3. execute the command via the ssh connection.
			session, err := conn.client.NewSession()
			if err != nil {
				// session 打开失败说明连接断了，重新连接重试一次。
				_ = conn.Close()
				_ = nRT.RecourceDelete(id)
				conn, err = newSSHConn(host, port, user, password)
				if err != nil {
					return fmt.Errorf("Exec--ExecSSH: failed to reconnect ssh: %w", err)
				}
				nRT.RecourceSet(conn)
				session, err = conn.client.NewSession()
				if err != nil {
					return fmt.Errorf("Exec--ExecSSH: failed to open session: %w", err)
				}
			}
			defer session.Close()

			var logBuf lockedBuffer
			session.Stdout = &logBuf
			session.Stderr = &logBuf // 同一 buffer → stdout/stderr 按到达顺序交错

			// 直接运行完整 shell 命令（可中断：interrupt 时关 session 终止远程命令）。
			runErr := make(chan error, 1)
			go func() { runErr <- session.Run(command) }()
			select {
			case err = <-runErr:
			case <-nRT.GetRtCtx().Done():
				_ = session.Close()
				return nRT.GetRtCtx().Err()
			}
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*ssh.ExitError); ok {
					exitCode = exitErr.ExitStatus()
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
