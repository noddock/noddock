package graph

import (
	"context"
	"encoding/json"
	"log"

	"github.com/dop251/goja"
)

// 用户 JS 脚本执行器，脚本通过注入的 GetValue/SetValue 访问节点参数。
type GojaExecutor struct {
	vm    *goja.Runtime
	rtCtx context.Context
	jsErr error
	done  chan struct{} // Close 时关闭，让 interrupt goroutine 退出
}

func (rt *GojaExecutor) GetErr() error {
	return rt.jsErr
}

// 创建 goja VM，rtCtx 取消时中断脚本执行，并注入 console.log。
func NewGojaExecutor(rtCtx context.Context) (*GojaExecutor, error) {
	vm := goja.New()
	done := make(chan struct{})
	go func() {
		select {
		case <-rtCtx.Done():
			vm.Interrupt("workflow cancelled")
		case <-done:
		}
	}()
	rt := &GojaExecutor{vm: vm, rtCtx: rtCtx, done: done}
	rt.vm.Set("console", map[string]func(goja.FunctionCall) goja.Value{
		"log": func(fc goja.FunctionCall) goja.Value {
			args := make([]any, len(fc.Arguments))
			for i, v := range fc.Arguments {
				args[i] = v.Export()
			}
			log.Println("[JS]", args)
			return goja.Undefined()
		},
	})
	return rt, nil
}

// 注入全局函数 SetValue（按参数 Kind 转换后写入 ctx）和 GetValue（JSON 参数解码为对象返回）。
func (rt *GojaExecutor) Prepare(ctx *Context, pMap map[string]*Parameter) error {
	rt.vm.Set("SetValue", func(paramID string, value any) {
		if p, ok := pMap[paramID]; ok {
			v, err := p.ValidateAndCast(value)
			if err != nil {
				rt.jsErr = err
				return
			}
			ctx.Set(paramID, v)
		}
	})
	rt.vm.Set("GetValue", func(paramID string) goja.Value {
		p, ok := pMap[paramID]
		if !ok {
			return goja.Undefined()
		}
		val := ctx.Get(paramID)
		if p.Kind == ParamJSON {
			if raw, ok := val.(json.RawMessage); ok {
				var decoded any
				if err := json.Unmarshal(raw, &decoded); err == nil {
					return rt.vm.ToValue(decoded)
				}
			}
		}
		return rt.vm.ToValue(val)
	})
	return nil
}

// 直接执行整段脚本，返回执行错误。
func (rt *GojaExecutor) ExecuteCode(code string) error {
	_, err := rt.vm.RunString(code)
	return err
}

// 停止 interrupt goroutine。
func (rt *GojaExecutor) Close() {
	close(rt.done)
}
