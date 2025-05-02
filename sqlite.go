package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	_ "github.com/mattn/go-sqlite3"
)

// SQLiteManager 管理SQLite数据库操作的结构体
type SQLiteManager struct {
	db *sql.DB
}

// NewSQLiteManager 创建新的SQLiteManager实例
func NewSQLiteManager(dbPath string) (*SQLiteManager, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Printf("[SQLite] 打开数据库失败: %v", err)
		return nil, fmt.Errorf("打开SQLite数据库失败: %v", err)
	}
	log.Printf("[SQLite] 成功打开数据库: %s", dbPath)
	return &SQLiteManager{db: db}, nil
}

// Close 关闭数据库连接
func (sm *SQLiteManager) Close() error {
	err := sm.db.Close()
	if err != nil {
		log.Printf("[SQLite] 关闭数据库连接失败: %v", err)
	} else {
		log.Println("[SQLite] 数据库连接已关闭")
	}
	return err
}

// Execute 执行SQL语句
func (sm *SQLiteManager) Execute(query string, args ...interface{}) (sql.Result, error) {
	log.Printf("[SQLite] 执行SQL语句: %s", query)
	result, err := sm.db.Exec(query, args...)
	if err != nil {
		log.Printf("[SQLite] 执行SQL失败: %v", err)
	} else {
		rowsAffected, _ := result.RowsAffected()
		log.Printf("[SQLite] 执行成功，影响行数: %d", rowsAffected)
	}
	return result, err
}

// Query 执行查询语句
func (sm *SQLiteManager) Query(query string, args ...interface{}) (*sql.Rows, error) {
	log.Printf("[SQLite] 执行查询: %s", query)
	rows, err := sm.db.Query(query, args...)
	if err != nil {
		log.Printf("[SQLite] 查询失败: %v", err)
	} else {
		log.Println("[SQLite] 查询执行成功")
	}
	return rows, err
}

// QueryRow 执行单行查询
func (sm *SQLiteManager) QueryRow(query string, args ...interface{}) *sql.Row {
	log.Printf("[SQLite] 执行单行查询: %s", query)
	return sm.db.QueryRow(query, args...)
}

// 注册SQLite工具
func registerSQLiteTools(mcpServer *server.MCPServer) {
	mcpServer.AddTool(mcp.NewTool("sqlite_execute",
		mcp.WithDescription("执行SQL语句"),
		mcp.WithString("db_path", mcp.Required(), mcp.Description("SQLite数据库路径")),
		mcp.WithString("query", mcp.Required(), mcp.Description("要执行的SQL语句"))),
		sqliteExecuteHandler)

	mcpServer.AddTool(mcp.NewTool("sqlite_query",
		mcp.WithDescription("执行查询语句"),
		mcp.WithString("db_path", mcp.Required(), mcp.Description("SQLite数据库路径")),
		mcp.WithString("query", mcp.Required(), mcp.Description("要执行的查询语句"))),
		sqliteQueryHandler)

	mcpServer.AddTool(mcp.NewTool("sqlite_query_row",
		mcp.WithDescription("执行单行查询"),
		mcp.WithString("db_path", mcp.Required(), mcp.Description("SQLite数据库路径")),
		mcp.WithString("query", mcp.Required(), mcp.Description("要执行的单行查询语句"))),
		sqliteQueryRowHandler)
}

// sqliteExecuteHandler 处理sqlite_execute工具调用
func sqliteExecuteHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dbPath := request.Params.Arguments["db_path"].(string)
	query := request.Params.Arguments["query"].(string)

	sm, err := NewSQLiteManager(dbPath)
	if err != nil {
		return nil, err
	}
	defer sm.Close()

	result, err := sm.Execute(query)
	if err != nil {
		return nil, err
	}

	rowsAffected, _ := result.RowsAffected()
	return mcp.NewToolResultText(fmt.Sprintf("执行成功，影响行数: %d", rowsAffected)), nil
}

// sqliteQueryHandler 处理sqlite_query工具调用
func sqliteQueryHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dbPath := request.Params.Arguments["db_path"].(string)
	query := request.Params.Arguments["query"].(string)

	sm, err := NewSQLiteManager(dbPath)
	if err != nil {
		return nil, err
	}
	defer sm.Close()

	rows, err := sm.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		pointers := make([]interface{}, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}

		err := rows.Scan(pointers...)
		if err != nil {
			return nil, err
		}

		rowData := make(map[string]interface{})
		for i, col := range columns {
			rowData[col] = values[i]
		}
		results = append(results, rowData)
	}

	return mcp.NewToolResultText(fmt.Sprintf("查询结果:\n%v", results)), nil
}

// sqliteQueryRowHandler 处理sqlite_query_row工具调用
func sqliteQueryRowHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dbPath := request.Params.Arguments["db_path"].(string)
	query := request.Params.Arguments["query"].(string)

	sm, err := NewSQLiteManager(dbPath)
	if err != nil {
		return nil, err
	}
	defer sm.Close()

	row := sm.QueryRow(query)

	columns := strings.Split(query, " ")[1:] // 简单解析SELECT后的列名
	values := make([]interface{}, len(columns))
	pointers := make([]interface{}, len(columns))
	for i := range values {
		pointers[i] = &values[i]
	}

	err = row.Scan(pointers...)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	for i, col := range columns {
		result[col] = values[i]
	}

	return mcp.NewToolResultText(fmt.Sprintf("单行查询结果:\n%v", result)), nil
}
