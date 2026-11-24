package manager

import (
	"sync"

	"github.com/noddock/noddock/runtime/graph"
)

// 全局节点注册表（ID → NodeSpec），节点包在 init() 里直接 Register 填充。
var (
	Registry = map[string]*graph.NodeSpec{}
	Resource = map[string]graph.Resource{}
	mu       sync.RWMutex
)

// 注册一个 NodeSpec（调用 Init 补全，重复 ID 覆盖）。实际中自定义节点ID无法覆盖官方节点。
func Register(spec *graph.NodeSpec) {
	mu.Lock()
	defer mu.Unlock()
	spec.Init()
	Registry[spec.ID] = spec
}
