package nodes

import (
	"math"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {

	param_float := graph.Parameter{
		ID:      "float",
		Kind:    graph.ParamFloat,
		Default: float64(0),
		Connect: []graph.ParamConnect{graph.ConnInput},
	}

	param_method := graph.Parameter{
		ID:         "method",
		Kind:       graph.ParamString,
		Default:    "floor",
		Selections: []string{"ceil", "floor", "round"},
	}

	param_int := graph.Parameter{
		ID:      "int",
		Kind:    graph.ParamInt,
		Default: 0,
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Core--Format--Float2Int",
		Name:        "Float to Int",
		Description: "Converts a float to an int. accepts ceil / floor / round.",
		Plist: []*graph.Parameter{
			&param_float,
			&param_method,
			&param_int,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			float := ctx.Get(param_float.ID).(float64)
			method := ctx.Get(param_method.ID).(string)
			var i int
			switch method {
			case "ceil":
				i = int(math.Ceil(float))
			case "round":
				i = int(math.Round(float))
			default:
				i = int(math.Floor(float))
			}
			ctx.Set(param_int.ID, i)
			return nil
		},
	})
}
