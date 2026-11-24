package graph

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// stderr 收集上限（1MB，防常驻进程日志无界增长）。
const terminalStderrCap = 1 << 20

// 常驻进程资源（cmd + stdin/stdout/stderr 管道），由 terminal 节点创建并存进资源表，其他节点可以通过 key 取到后按约定协议通信。
// 当前以 unix build 做展示
type Terminal struct {
	Cmd    *exec.Cmd
	Stdin  io.WriteCloser
	Stdout *bufio.Reader
	Stderr *bytes.Buffer // 后台 goroutine 收集（带上限）

	id     string
	mu     sync.Mutex
	wmu    sync.Mutex // 串行化 stdin 写
	closed bool
	done   chan struct{} // 进程退出时关闭
}

// Resource 要求
func (t *Terminal) ID() string            { return t.id }
func (t *Terminal) Done() <-chan struct{} { return t.done }

// 创建绑定 ctx 的 exec.Cmd，ctx 取消时杀进程。
func NewCmd(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

// 创建独立进程（不经 shell，name + args 直接 exec，工作目录 dir，防止路径含空格或中文字符以及注入攻击）
func NewCmdFull(ctx context.Context, dir, name string, args ...string) *exec.Cmd {
	cmd := NewCmd(ctx, name, args...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd
}

// 创建 sh -c 命令（工作目录 dir，独立进程组便于整树 kill）。
func NewShellCmd(ctx context.Context, command, dir string) *exec.Cmd {
	cmd := NewCmd(ctx, "sh", "-c", command)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd
}

// 把已创建的 *exec.Cmd 包装成常驻 Terminal（stdin/stdout/stderr 管道 + done 通道 + stderr 收集），NewTerminal 与 NewTerminalDirect 共用。
func newTerminalFromCmd(cmd *exec.Cmd, id string) (*Terminal, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()

	stderrBuf := &bytes.Buffer{}
	go func() {
		buf := make([]byte, 4096)
		total := 0
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				total += n
				if total <= terminalStderrCap {
					stderrBuf.Write(buf[:n])
				}
			}
			if err != nil {
				return
			}
		}
	}()

	return &Terminal{
		Cmd:    cmd,
		Stdin:  stdin,
		Stdout: bufio.NewReader(stdout),
		Stderr: stderrBuf,
		id:     id,
		done:   done,
	}, nil
}

// 以 shell 启动常驻进程（sh -c，工作目录 dir），后台收集 stderr，返回资源。
func NewTerminal(dir, command, id string) (*Terminal, error) {
	cmd := NewShellCmd(context.Background(), command, dir)
	return newTerminalFromCmd(cmd, id)
}

// 以独立进程启动常驻进程（不经 shell，name + args 直接 exec，工作目录 dir），路径含空格时比 shell 拼接更稳健。
func NewTerminalDirect(dir, name string, args []string, id string) (*Terminal, error) {
	cmd := NewCmdFull(context.Background(), dir, name, args...)
	return newTerminalFromCmd(cmd, id)
}

// 把 v 序列化成一行 JSON 写入 stdin，串行化保证多 goroutine 并发安全。
func (t *Terminal) WriteLine(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	t.wmu.Lock()
	defer t.wmu.Unlock()
	_, err = t.Stdin.Write(data)
	return err
}

// 销毁进程：关 stdin 触发 EOF → 500ms 宽限 → kill 整个进程组，幂等。
func (t *Terminal) Close() error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	t.mu.Unlock()

	_ = t.Stdin.Close()
	select {
	case <-t.done:
		return nil
	case <-time.After(500 * time.Millisecond):
	}
	if t.Cmd.Process != nil {
		killTerminalProcessGroup(t.Cmd.Process.Pid)
		_ = t.Cmd.Process.Kill()
	}
	return nil
}

// 报告进程是否还在运行。
func (t *Terminal) Alive() bool {
	if t.Cmd.Process == nil {
		return false
	}
	return processAlive(t.Cmd.Process.Pid)
}

// 向 -pid 发 SIGKILL，杀掉独立进程组里的整棵进程树。
func killTerminalProcessGroup(pid int) {
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}

// 用 signal 0 探测进程是否存在。
func processAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

// mock 实现：unix 下 shell 输出默认 UTF-8，原样返回（返回 mock 结果）。
// 主要用于非unix下中文字符显示乱码的问题。
func FixEncoding(b []byte) []byte {
	return b
}
