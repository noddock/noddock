package nodes

import (
	"fmt"

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

	param_target := graph.Parameter{
		ID:      "target",
		Kind:    graph.ParamString,
		Default: "",
		Connect: []graph.ParamConnect{graph.ConnEdit},
	}

	manager.Register(&graph.NodeSpec{
		ID:          "Logic--RunWorkflow",
		Name:        "Run Workflow",
		Description: `Runs a target workflow. \n if this node in target workflow, its async must be true.`,
		Plist: []*graph.Parameter{
			&param_then,
			&param_target,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			target := ctx.Get(param_target.ID).(string)
			if target == "" {
				return fmt.Errorf("Logic--RunWorkflow: target is empty, specify the workflow ID to run")
			}
			if err := nRT.RunWorkflow(target); err != nil {
				return err
			}
			ctx.Set(param_then.ID, "activate")
			return nil
		},
		InstProperty: &graph.InstProperty{
			Async: true,
		},
	})
}
