package nodes

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

// dbConn 是资源表里存的活数据库连接（原 repo.GeneralRepo 的逻辑搬进来）。
type dbConn struct {
	db *sql.DB
	id string // 资源标识（= 资源表 key）
}

// ID 返回资源标识（= 资源表 key）。
func (c *dbConn) ID() string { return c.id }

// openDB 打开数据库（sqlite 先建父目录，再 sql.Open + Ping + PRAGMA）。
func openDB(dbType, dbAddr string) (*dbConn, error) {
	// sqlite：dbAddr 是文件路径，确保父目录存在（待创建的 sqlite 文件）。
	if dbType == "sqlite" {
		if dir := filepath.Dir(dbAddr); dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("openDB: failed to create dir: %w", err)
			}
		}
	}
	db, err := sql.Open(dbType, dbAddr)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	// sqlite 专用 PRAGMA（其他驱动会失败，所以只在 sqlite 时设）。
	if dbType == "sqlite" {
		if _, err := db.Exec(`
PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
PRAGMA foreign_keys=ON;
PRAGMA busy_timeout=5000;
`); err != nil {
			db.Close()
			return nil, err
		}
	}
	return &dbConn{db: db}, nil
}

func (c *dbConn) Close() error {
	return c.db.Close()
}

// query 执行查询 SQL，返回行（每行是 map[列名]值）。空结果返回空 slice（方便 JSON 序列化）。
func (c *dbConn) query(ctx context.Context, sqlText string, args []any) (map[string]any, error) {
	rows, err := c.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"columns": columns,
		"rows":    make([]map[string]any, 0),
	}

	for rows.Next() {
		dest := make([]any, len(columns))
		for i, ct := range colTypes {
			switch ct.DatabaseTypeName() {
			case "BLOB", "BINARY", "VARBINARY":
				dest[i] = new([]byte)
			default:
				dest[i] = new(any)
			}
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(columns))
		for i, col := range columns {
			switch v := dest[i].(type) {
			case *any:
				row[col] = *v
			case *[]byte:
				if v == nil {
					row[col] = nil
				} else {
					row[col] = *v
				}
			}
		}
		result["rows"] = append(result["rows"].([]map[string]any), row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// exec 执行写 SQL（CREATE/INSERT/UPDATE/DELETE 等），返回 last_insert_id + rows_affected。
func (c *dbConn) exec(ctx context.Context, sqlText string, args []any) (map[string]any, error) {
	res, err := c.db.ExecContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	lastInsertId, _ := res.LastInsertId()
	rowsAffected, _ := res.RowsAffected()
	return map[string]any{
		"last_insert_id": lastInsertId,
		"rows_affected":  rowsAffected,
	}, nil
}
