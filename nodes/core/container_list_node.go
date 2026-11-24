package nodes

import (
	"encoding/json"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
	"github.com/noddock/noddock/runtime/tool"
)

func init() {

	param_list := graph.Parameter{
		ID:      "list",
		Kind:    graph.ParamJSON,
		Default: "[]",
		Connect: []graph.ParamConnect{graph.ConnInput, graph.ConnEdit, graph.ConnOutput},
	}

	param_append := graph.Parameter{
		ID:      "append",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}
	param_popout := graph.Parameter{
		ID:      "popout",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Core--Container--List",
		Name:        "List",
		Description: `Append if the append input has content; popout if the popout output is connected. The two are independent (non-exclusive); append runs before popout.`,
		Plist: []*graph.Parameter{
			&param_list,
			&param_append,
			&param_popout,
		},

		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			listRaw := ctx.Get(param_list.ID).(json.RawMessage)
			var list []any
			if err := json.Unmarshal(listRaw, &list); err != nil {
				return err
			}

			if nRT.UpstreamCount(param_append.ID) > 0 {
				appendRaw := ctx.Get(param_append.ID).(json.RawMessage)
				var item any
				if err := json.Unmarshal(appendRaw, &item); err != nil {
					return err
				}
				list = append(list, item)
			}
			if nRT.DownstreamCount(param_popout.ID) > 0 && len(list) > 0 {
				out := list[len(list)-1]
				list = list[:len(list)-1]
				ctx.Set(param_popout.ID, tool.Any2JsonRaw(out))
			}

			listData, err := json.Marshal(list)
			if err != nil {
				return err
			}
			ctx.Set(param_list.ID, json.RawMessage(listData))
			return nil
		},
	})
}
