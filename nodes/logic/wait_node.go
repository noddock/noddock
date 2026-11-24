package nodes

import (
	"time"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {
	param_then := graph.Parameter{
		ID:      "then",
		Kind:    graph.ParamLogic,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnInput, graph.ConnOutput},
	}
	param_wait := graph.Parameter{
		ID:      "wait",
		Kind:    graph.ParamInt,
		Default: 1,
		Connect: []graph.ParamConnect{graph.ConnEdit},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Logic--Wait",
		Name:        "Wait",
		Description: "Delay by <wait> seconds.",
		Plist: []*graph.Parameter{
			&param_then,
			&param_wait,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			wait := ctx.Get(param_wait.ID).(int)

			timer := time.NewTimer(time.Second * time.Duration(wait))
			defer timer.Stop()

			rtCtx := nRT.GetRtCtx()
			select {
			case <-timer.C:
				ctx.Set(param_then.ID, "start")
				return nil
			case <-rtCtx.Done():
				return rtCtx.Err()
			}
		},
	})
}
