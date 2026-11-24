package nodes

import (
	"bytes"
	"sync"
)

// exec 系列节点的共享类型。

// ExecIn 是 exec 系列节点的统一输入（command + args）。
type ExecIn struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

// EnvironmentDescriptor 描述执行环境（docker 容器等）。
type EnvironmentDescriptor struct {
	Type      string `json:"type"`
	Container string `json:"container"`
}
