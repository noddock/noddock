package main

import (
	"encoding/json"
	"fmt"

	_ "github.com/noddock/noddock/nodes"
	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func main() {
	// 1. 加载节点
	node := manager.Registry["Json--InlineJS"]

	// 2.1 设置默认值
	ctx := graph.NewContext()
	for _, p := range node.Plist {
		if v, err := p.ValidateAndCast(p.Default); err == nil {
			ctx.Set(p.ID, v)
		}
	}
	// 2.2 设置输入或编辑变量（json 参数直接给 json.RawMessage，避免 Any2JsonRaw 对字符串二次编码）
	ctx.Set("input", json.RawMessage(`{"a":1, "b":"2", "c":[1,2,3]}`))
	ctx.Set("code", "output=input.c;")

	// 3. 运行
	if err := node.Run(ctx, &runTool{node}); err != nil {
		fmt.Println(err)
		return
	}

	// 4. 打印结果
	for k, v := range ctx.Data {
		fmt.Println(k, ":", v)
	}
}
