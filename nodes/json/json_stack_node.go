package nodes

import (
	"encoding/json"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {
	param_in1 := graph.Parameter{
		ID:      "in1",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}
	param_in2 := graph.Parameter{
		ID:      "in2",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}
	param_in3 := graph.Parameter{
		ID:      "in3",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}
	param_out := graph.Parameter{
		ID:      "out",
		Kind:    graph.ParamJSON,
		Default: "[]",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Json--JsonStack",
		Name:        "Json Stack",
		Description: `Combines 3 JSON inputs into one list (in order). Arrays are flattened; {} is skipped.`,
		Plist: []*graph.Parameter{
			&param_in1,
			&param_in2,
			&param_in3,
			&param_out,
		},
		Run: func(ctx *graph.Context, _ graph.NodeRunTool) error {
			var result []any

			for _, id := range []string{param_in1.ID, param_in2.ID, param_in3.ID} {
				raw, _ := ctx.Get(id).(json.RawMessage)
				if len(raw) == 0 || string(raw) == "null" {
					continue
				}

				// 检查是否为 {}（空对象）→ 跳过
				var probe map[string]any
				if err := json.Unmarshal(raw, &probe); err == nil {
					if len(probe) == 0 {
						continue // 空对象，跳过
					}
					// 是对象，作为单个元素 append
					result = append(result, probe)
					continue
				}

				// 尝试解析为数组 → 展开
				var arr []any
				if err := json.Unmarshal(raw, &arr); err == nil {
					result = append(result, arr...)
					continue
				}

				// 其他类型（数字/字符串/bool）→ 作为单个元素 append
				var v any
				if err := json.Unmarshal(raw, &v); err == nil {
					result = append(result, v)
				}
			}

			if result == nil {
				result = []any{}
			}
			out, err := json.Marshal(result)
			if err != nil {
				return err
			}
			ctx.Set(param_out.ID, json.RawMessage(out))
			return nil
		},
	})
}
