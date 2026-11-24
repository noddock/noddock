package nodes

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	_ "modernc.org/sqlite" // sqlite 驱动

	"github.com/noddock/noddock/runtime/graph"
	"github.com/noddock/noddock/runtime/manager"
	"github.com/noddock/noddock/runtime/tool"
	"github.com/noddock/noddock/runtime/tool/fo"
)

func init() {
	paramSqlText := graph.Parameter{
		ID:      "sql_text",
		Name:    "sql text",
		Kind:    graph.ParamString,
		Connect: []graph.ParamConnect{graph.ConnInput},
		Default: "",
	}
	paramArgs := graph.Parameter{
		ID:      "args",
		Kind:    graph.ParamJSON,
		Connect: []graph.ParamConnect{graph.ConnInput},
		Default: "[]",
	}
	paramMode := graph.Parameter{
		ID:         "mode",
		Kind:       graph.ParamString,
		Connect:    []graph.ParamConnect{graph.ConnInput},
		Default:    "none",
		Selections: []string{"query", "exec", "none"},
	}
	paramDbType := graph.Parameter{
		ID:         "dbtype",
		Kind:       graph.ParamString,
		Selections: []string{"sqlite"},
		Default:    "sqlite",
	}
	paramDbAddr := graph.Parameter{
		ID:      "dbAddr",
		Kind:    graph.ParamString,
		Default: "",
	}
	paramRes := graph.Parameter{
		ID:      "res",
		Name:    "result",
		Kind:    graph.ParamJSON,
		Connect: []graph.ParamConnect{graph.ConnOutput},
		Default: "{}",
	}

	manager.Register(&graph.NodeSpec{
		ID:          "DB--SQL",
		Name:        "SQL",
		Description: `Run sql code on <dbType> from <dbAddr>, mode: query|exec. Only local sqlite supported now, other drivers (postgres/mysql) will support if in need. \n Addr accepts relative path under this project (/.../noddock/project/<project_name>/<dbAddr>).`,
		Plist: []*graph.Parameter{
			&paramMode,
			&paramSqlText,
			&paramArgs,
			&paramDbType,
			&paramDbAddr,
			&paramRes,
		},
		Run: func(ctx *graph.Context, nRT graph.NodeRunTool) error {
			defer ctx.Set(paramMode.ID, "none")

			sqlText := ctx.Get(paramSqlText.ID).(string)
			argsJson := ctx.Get(paramArgs.ID).(json.RawMessage)
			mode := ctx.Get(paramMode.ID).(string)
			dbType := ctx.Get(paramDbType.ID).(string)
			dbAddr := ctx.Get(paramDbAddr.ID).(string)

			if mode == "none" {
				return nil
			}

			var args []any
			if len(argsJson) > 0 && string(argsJson) != "null" && string(argsJson) != "{}" {
				if err := json.Unmarshal(argsJson, &args); err != nil {
					return fmt.Errorf("DB--SQL: args must be a JSON array: %w", err)
				}
			}

			// 缺省 dbAddr：项目目录的 data.sqlite（绝对路径，与原 GetRepo 一致）。
			if dbAddr == "" {
				dbAddr = filepath.Join(nRT.GetProjectDir(), "data.sqlite")
				dbType = "sqlite"
			} else if dbType == "sqlite" {
				var err error
				dbAddr, err = fo.GetPath(dbAddr, nRT.GetPID(), false, true)
				if err != nil {
					return err
				}
			}

			// 取或建连接（存资源表，跨 run 存活；key=ID）。
			id := "db:" + dbAddr
			var conn *dbConn
			if nRT.RecourceHas(id) {
				if v, err := nRT.RecourceGet(id); err == nil {
					conn, _ = v.(*dbConn)
				}
			}
			if conn == nil {
				var err error
				conn, err = openDB(dbType, dbAddr)
				if err != nil {
					return err
				}
				conn.id = id
				nRT.RecourceSet(conn)
			}

			var out map[string]any
			var err error
			switch mode {
			case "query":
				out, err = conn.query(nRT.GetRtCtx(), sqlText, args)
			case "exec":
				out, err = conn.exec(nRT.GetRtCtx(), sqlText, args)
			default:
				return fmt.Errorf("DB--SQL: mode must be query or exec, got %q", mode)
			}

			if err != nil {
				ctx.Set(paramRes.ID, tool.Any2JsonRaw(map[string]any{"status": "error", "error": err.Error()}))
				return nil
			}

			out["status"] = "ok"
			ctx.Set(paramRes.ID, tool.Any2JsonRaw(out))
			return nil
		},
	})
}
