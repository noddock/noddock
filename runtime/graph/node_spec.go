package graph

import (
	"context"
)

const (
	PermissionSafe      = "safe"
	PermissionRisk      = "risk"
	PermissionDangerous = "dangerous"
)

// 节点定义
type NodeSpec struct {
	ID          string
	Name        string
	Description string

	RequiredPermission string

	Parameters map[string]*Parameter
	Plist      []*Parameter
	Run        func(ctx *Context, nRT NodeRunTool) error

	ExecProperty *ExecProperty
	InstProperty *InstProperty
}

// 节点运行时可用的全部能力，由运行时注入实现，节点通过它访问资源/项目目录/vault 等。
type NodeRunTool interface {
	NotifySiblingsNewStream(nodeKey, streamName string)
	GetPID() string
	GetNID() string
	GetNIID() string
	GetRtCtx() context.Context
	GetProjectDir() string
	RunWorkflow(string) error

	RecourceHas(key string) bool
	RecourceSet(res Resource)
	RecourceGet(key string) (Resource, error)
	RecourceDelete(key string) error

	DownstreamCount(paramID string) int
	UpstreamCount(paramID string) int

	VaultLookup(ref string) (string, error)
}

// 无connect则默认edit
func (ns *NodeSpec) Init() {
	if ns.ID == "" {
		return
	}

	ns.Parameters = make(map[string]*Parameter)
	for i, param := range ns.Plist {
		param.Order = i
		if param.Connect == nil {
			param.Connect = []ParamConnect{ConnEdit}
		}
		if param.Kind == ParamStream {
			param.Default = NewStream[string]()
		}
		if param.Kind == ParamBool {
			param.Selections = []bool{true, false}
		}
		ns.Parameters[param.ID] = param
	}
}

// 默认的模板，可在assets/templates查看
type ExecProperty struct {
	CustomRenderPath string
}

type InstProperty struct {
	Async bool
}
