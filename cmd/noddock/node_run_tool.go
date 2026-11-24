package main

import (
	"context"
	"fmt"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

// mock run tool
type runTool struct {
	spec *graph.NodeSpec
}

var DownstreamNum int

// 通知当前节点及其复制的节点，在自定义HTML里建立Stream连接，准备接收信息流
func (t *runTool) NotifySiblingsNewStream(nodeKey, streamName string) {}

func (t *runTool) GetPID() string  { return "example_project" }
func (t *runTool) GetNID() string  { return t.spec.ID + "_I1_E1" }
func (t *runTool) GetNIID() string { return t.spec.ID + "_I1" }

// 返回运行 context，随用户主动cancel时取消
func (t *runTool) GetRtCtx() context.Context { return context.Background() }

// 返回绝对路径
func (t *runTool) GetProjectDir() string { return "example_project" }

// 用于 RunWorkflow Node，将指定工作流的节点加入运行队列，会等待到指定工作流内全部节点运行结束。
// 当指定工作流包含自己时，需要开启async，避免自己永远等不到自己结束
func (t *runTool) RunWorkflow(string) error {
	return nil
}

// 报告资源表里是否有该 key。Resource给定key后可以跨节点获取，避免同一个资源每次都重新加载
func (t *runTool) RecourceHas(key string) bool {
	_, ok := manager.Resource[key]
	return ok
}

// 把资源按 ID 存入资源表。
func (t *runTool) RecourceSet(res graph.Resource) {
	manager.Resource[res.ID()] = res
}

// 按 key 取资源，不存在时返回 error。
func (t *runTool) RecourceGet(key string) (graph.Resource, error) {
	res, ok := manager.Resource[key]
	if !ok {
		return nil, fmt.Errorf("resource %q not found", key)
	}
	return res, nil
}

// 从资源表移除 key，不存在时返回 error。
func (t *runTool) RecourceDelete(key string) error {
	if _, ok := manager.Resource[key]; !ok {
		return fmt.Errorf("resource %q not found", key)
	}
	delete(manager.Resource, key)
	return nil
}

// 返回当前变量的下游边数量，常用于stream变量统计下游stream变量的数量，等待全部都准备完毕后开始发送数据
func (t *runTool) DownstreamCount(paramID string) int { return 0 }

// 返回当前变量的下游边数量，常用于根据是否有输入变量需要调整方法的节点，比如 List Node
func (t *runTool) UpstreamCount(paramID string) int { return DownstreamNum }

// 从保险箱获取API，避免API或密码等私人信息暴露在项目文件和节点中，原则上只有特定节点的特定环节需要使用
func (t *runTool) VaultLookup(ref string) (string, error) {
	if ref == "my-api-key" {
		return "sk-sUpErcRAzytHurSdAyaNDvmE50", nil
	} else {
		return ref, nil
	}
}
