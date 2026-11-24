package nodes

import (
	"fmt"

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
)

func init() {

	param_input := graph.Parameter{
		ID:      "input",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnInput},
	}

	param_code := graph.Parameter{
		ID:      "code",
		Kind:    graph.ParamString,
		Default: "",
	}

	param_output := graph.Parameter{
		ID:      "output",
		Kind:    graph.ParamJSON,
		Default: "{}",
		Connect: []graph.ParamConnect{graph.ConnOutput},
	}
	pMap := map[string]*graph.Parameter{
		param_input.ID:  &param_input,
		param_code.ID:   &param_code,
		param_output.ID: &param_output,
	}
	manager.Register(&graph.NodeSpec{
		ID:                 "Json--InlineJS",
		RequiredPermission: graph.PermissionRisk,
		Name:               "Inline JS",
		Description:        "Inline JavaScript execution to edit intput to output. code eg: output=input.text",
		Plist: []*graph.Parameter{
			&param_input,
			&param_code,
			&param_output,
		},

		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			code := ctx.Get(param_code.ID).(string)
			fullcode := fmt.Sprintf(`
const input = GetValue("input");
%s;
output = output ?? {};
SetValue("output", output);
				`, code)
			rt, err := graph.NewGojaExecutor(nRT.GetRtCtx())
			if err != nil {
				return err
			}
			defer rt.Close()

			if err := rt.Prepare(ctx, pMap); err != nil {
				return err
			}
			if err := rt.ExecuteCode(string(fullcode)); err != nil {
				return err
			}

			if rt.GetErr() != nil {
				return rt.GetErr()
			}
			return nil
		},
	})
}
