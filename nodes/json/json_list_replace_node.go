package nodes

import (
	"encoding/json"
	"errors"
	"slices"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
	"github.com/noddock/noddock/runtime/tool"
)

func init() {

	param_list := graph.Parameter{
		ID:      "list",
		Kind:    graph.ParamJSON,
		Default: "[]",
		Connect: []graph.ParamConnect{graph.ConnInput, graph.ConnOutput},
	}

	param_start := graph.Parameter{
		ID:      "start",
		Kind:    graph.ParamInt,
		Default: 0,
	}

	param_end := graph.Parameter{
		ID:      "end",
		Kind:    graph.ParamInt,
		Default: 0,
	}

	param_replace := graph.Parameter{
		ID:      "replace",
		Kind:    graph.ParamJSON,
		Default: "[]",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Json--JsonListReplace",
		Name:        "Json List Replace",
		Description: `Replaces a slice of a JSON array with another array. replace must be a list.`,
		Plist: []*graph.Parameter{
			&param_list,
			&param_start,
			&param_end,
			&param_replace,
		},

		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			listRaw := ctx.Get(param_list.ID).(json.RawMessage)
			replaceRaw := ctx.Get(param_replace.ID).(json.RawMessage)
			start := ctx.Get(param_start.ID).(int)
			end := ctx.Get(param_end.ID).(int)
			var list []any
			if err := json.Unmarshal(listRaw, &list); err != nil {
				return err
			}
			length := len(list)
			if end < 0 {
				end = length + end
			}
			// 负 start 不支持（故意）；slices.Replace 对越界 [start,end) 会 panic，这里先校验返回友好错误
			if start < 0 || end < 0 || start > end || end > length {
				return errors.New("bad index")
			}

			var repalce []any
			if err := json.Unmarshal(replaceRaw, &repalce); err != nil {
				return err
			}

			list = slices.Replace(list, start, end, repalce...)
			ctx.Set(param_list.ID, tool.Any2JsonRaw(list))
			return nil
		},
	})
}
