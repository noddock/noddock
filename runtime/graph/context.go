package graph

import (
	"sync"
)

// Context 节点参数值的扁平命名空间，键是参数 ID，节点通过它读写自己的参数。
type Context struct {
	mu   sync.RWMutex
	Data map[string]any
}

func (c *Context) Get(key string) any {
	return c.Data[key]
}

func (c *Context) Set(key string, val any) {
	c.Data[key] = val
}

func NewContext() *Context {
	return &Context{
		Data: map[string]any{},
	}
}
