package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// 数据库连接管理
var (
	mysqlDB    *sql.DB
	postgresDB *sql.DB
	mysqlTx    *sql.Tx
	postgresTx *sql.Tx
)

// 添加数据库工具函数
func addDatabaseTools(mcpServer *server.MCPServer) {
	// MySQL连接
	mcpServer.AddTool(mcp.NewTool("mysql_connect",
		mcp.WithDescription("连接到MySQL数据库"),
		mcp.WithString("dsn", mcp.Required(), mcp.Description("MySQL连接字符串(示例: user:password@tcp(localhost:3306)/dbname)"))),
		connectMySQL)

	// PostgreSQL连接
	mcpServer.AddTool(mcp.NewTool("postgres_connect",
		mcp.WithDescription("连接到PostgreSQL数据库"),
		mcp.WithString("dsn", mcp.Required(), mcp.Description("PostgreSQL连接字符串(示例: postgres://user:password@localhost:5432/dbname?sslmode=disable)"))),
		connectPostgres)

	// 执行查询
	mcpServer.AddTool(mcp.NewTool("db_query",
		mcp.WithDescription("执行SQL查询"),
		mcp.WithString("query", mcp.Required(), mcp.Description("要执行的SQL查询")),
		mcp.WithString("db_type", mcp.Required(), mcp.Description("数据库类型(mysql/postgres)"))),
		executeQuery)

	// 关闭连接
	mcpServer.AddTool(mcp.NewTool("db_close",
		mcp.WithDescription("关闭数据库连接"),
		mcp.WithString("db_type", mcp.Required(), mcp.Description("数据库类型(mysql/postgres)"))),
		closeConnection)

	// 事务管理
	mcpServer.AddTool(mcp.NewTool("db_begin_tx",
		mcp.WithDescription("开始数据库事务"),
		mcp.WithString("db_type", mcp.Required(), mcp.Description("数据库类型(mysql/postgres)"))),
		beginTransaction)

	mcpServer.AddTool(mcp.NewTool("db_commit_tx",
		mcp.WithDescription("提交数据库事务"),
		mcp.WithString("db_type", mcp.Required(), mcp.Description("数据库类型(mysql/postgres)"))),
		commitTransaction)

	mcpServer.AddTool(mcp.NewTool("db_rollback_tx",
		mcp.WithDescription("回滚数据库事务"),
		mcp.WithString("db_type", mcp.Required(), mcp.Description("数据库类型(mysql/postgres)"))),
		rollbackTransaction)

	// 批量操作
	mcpServer.AddTool(mcp.NewTool("db_batch_insert",
		mcp.WithDescription("批量插入数据"),
		mcp.WithString("table", mcp.Required(), mcp.Description("表名")),
		mcp.WithString("data", mcp.Required(), mcp.Description("JSON格式的数据数组")),
		mcp.WithString("db_type", mcp.Required(), mcp.Description("数据库类型(mysql/postgres)"))),
		batchInsert)

	// 表结构查询
	mcpServer.AddTool(mcp.NewTool("db_get_schema",
		mcp.WithDescription("获取表结构信息"),
		mcp.WithString("table", mcp.Required(), mcp.Description("表名")),
		mcp.WithString("db_type", mcp.Required(), mcp.Description("数据库类型(mysql/postgres)"))),
		getTableSchema)
}

// 连接MySQL
func connectMySQL(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dsn := request.Params.Arguments["dsn"].(string)
	log.Printf("正在连接MySQL数据库，DSN: %s", dsn)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Printf("MySQL连接失败: %v", err)
		return nil, fmt.Errorf("MySQL连接失败: %v", err)
	}
	mysqlDB = db
	log.Println("MySQL连接成功")
	return mcp.NewToolResultText("MySQL连接成功"), nil
}

// 连接PostgreSQL
func connectPostgres(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dsn := request.Params.Arguments["dsn"].(string)
	log.Printf("正在连接PostgreSQL数据库，DSN: %s", dsn)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Printf("PostgreSQL连接失败: %v", err)
		return nil, fmt.Errorf("PostgreSQL连接失败: %v", err)
	}
	postgresDB = db
	log.Println("PostgreSQL连接成功")
	return mcp.NewToolResultText("PostgreSQL连接成功"), nil
}

// 执行查询
func executeQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := request.Params.Arguments["query"].(string)
	dbType := request.Params.Arguments["db_type"].(string)
	log.Printf("执行SQL查询，数据库类型: %s, 查询语句: %s", dbType, query)

	var db *sql.DB
	switch dbType {
	case "mysql":
		db = mysqlDB
	case "postgres":
		db = postgresDB
	default:
		log.Printf("不支持的数据库类型: %s", dbType)
		return nil, errors.New("不支持的数据库类型")
	}

	if db == nil {
		log.Println("数据库未连接，请先调用连接函数")
		return nil, errors.New("数据库未连接，请先调用连接函数")
	}

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("查询执行失败: %v", err)
		return nil, fmt.Errorf("查询执行失败: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("获取列名失败: %v", err)
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
			return nil, fmt.Errorf("扫描行失败: %v", err)
		}

		result := make(map[string]interface{})
		for i, col := range columns {
			result[col] = values[i]
		}
		results = append(results, result)
	}

	resultData, err := json.Marshal(results)
	if err != nil {
		return nil, fmt.Errorf("JSON编码失败: %v", err)
	}

	return mcp.NewToolResultText(string(resultData)), nil
}

// 开始事务
func beginTransaction(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dbType := request.Params.Arguments["db_type"].(string)
	log.Printf("正在开始%s事务", dbType)

	var err error
	switch dbType {
	case "mysql":
		if mysqlDB == nil {
			log.Println("MySQL数据库未连接")
			return nil, errors.New("MySQL数据库未连接")
		}
		mysqlTx, err = mysqlDB.Begin()
	case "postgres":
		if postgresDB == nil {
			log.Println("PostgreSQL数据库未连接")
			return nil, errors.New("PostgreSQL数据库未连接")
		}
		postgresTx, err = postgresDB.Begin()
	default:
		log.Printf("不支持的数据库类型: %s", dbType)
		return nil, errors.New("不支持的数据库类型")
	}

	if err != nil {
		log.Printf("开始%s事务失败: %v", dbType, err)
		return nil, fmt.Errorf("开始事务失败: %v", err)
	}
	log.Printf("%s事务已开始", dbType)
	return mcp.NewToolResultText("事务已开始"), nil
}

// 提交事务
func commitTransaction(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dbType := request.Params.Arguments["db_type"].(string)
	log.Printf("正在提交%s事务", dbType)

	switch dbType {
	case "mysql":
		if mysqlTx == nil {
			log.Println("MySQL事务未开始")
			return nil, errors.New("MySQL事务未开始")
		}
		if err := mysqlTx.Commit(); err != nil {
			log.Printf("提交MySQL事务失败: %v", err)
			return nil, fmt.Errorf("提交事务失败: %v", err)
		}
		mysqlTx = nil
	case "postgres":
		if postgresTx == nil {
			log.Println("PostgreSQL事务未开始")
			return nil, errors.New("PostgreSQL事务未开始")
		}
		if err := postgresTx.Commit(); err != nil {
			log.Printf("提交PostgreSQL事务失败: %v", err)
			return nil, fmt.Errorf("提交事务失败: %v", err)
		}
		postgresTx = nil
	default:
		log.Printf("不支持的数据库类型: %s", dbType)
		return nil, errors.New("不支持的数据库类型")
	}

	log.Printf("%s事务已提交", dbType)
	return mcp.NewToolResultText("事务已提交"), nil
}

// 回滚事务
func rollbackTransaction(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dbType := request.Params.Arguments["db_type"].(string)
	log.Printf("正在回滚%s事务", dbType)

	switch dbType {
	case "mysql":
		if mysqlTx == nil {
			log.Println("MySQL事务未开始")
			return nil, errors.New("MySQL事务未开始")
		}
		if err := mysqlTx.Rollback(); err != nil {
			log.Printf("回滚MySQL事务失败: %v", err)
			return nil, fmt.Errorf("回滚事务失败: %v", err)
		}
		mysqlTx = nil
	case "postgres":
		if postgresTx == nil {
			log.Println("PostgreSQL事务未开始")
			return nil, errors.New("PostgreSQL事务未开始")
		}
		if err := postgresTx.Rollback(); err != nil {
			log.Printf("回滚PostgreSQL事务失败: %v", err)
			return nil, fmt.Errorf("回滚事务失败: %v", err)
		}
		postgresTx = nil
	default:
		log.Printf("不支持的数据库类型: %s", dbType)
		return nil, errors.New("不支持的数据库类型")
	}

	log.Printf("%s事务已回滚", dbType)
	return mcp.NewToolResultText("事务已回滚"), nil
}

// 批量插入
func batchInsert(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	table := request.Params.Arguments["table"].(string)
	data := request.Params.Arguments["data"].(string)
	dbType := request.Params.Arguments["db_type"].(string)

	var db *sql.DB
	switch dbType {
	case "mysql":
		db = mysqlDB
	case "postgres":
		db = postgresDB
	default:
		return nil, errors.New("不支持的数据库类型")
	}

	if db == nil {
		return nil, errors.New("数据库未连接")
	}

	var records []map[string]interface{}
	if err := json.Unmarshal([]byte(data), &records); err != nil {
		return nil, fmt.Errorf("解析JSON数据失败: %v", err)
	}

	if len(records) == 0 {
		return nil, errors.New("没有数据需要插入")
	}

	// 获取列名
	columns := make([]string, 0, len(records[0]))
	for col := range records[0] {
		columns = append(columns, col)
	}

	// 构建批量插入SQL
	var query string
	var args []interface{}
	if dbType == "mysql" {
		query = fmt.Sprintf("INSERT INTO %s (%s) VALUES ", table, strings.Join(columns, ", "))
		placeholders := make([]string, 0, len(records))
		for _, record := range records {
			ph := make([]string, 0, len(columns))
			for _, col := range columns {
				args = append(args, record[col])
				ph = append(ph, "?")
			}
			placeholders = append(placeholders, "("+strings.Join(ph, ", ")+")")
		}
		query += strings.Join(placeholders, ", ")
	} else {
		query = fmt.Sprintf("INSERT INTO %s (%s) VALUES ", table, strings.Join(columns, ", "))
		placeholders := make([]string, 0, len(records))
		for i, record := range records {
			ph := make([]string, 0, len(columns))
			for j, col := range columns {
				args = append(args, record[col])
				ph = append(ph, fmt.Sprintf("$%d", i*len(columns)+j+1))
			}
			placeholders = append(placeholders, "("+strings.Join(ph, ", ")+")")
		}
		query += strings.Join(placeholders, ", ")
	}

	// 执行批量插入
	_, err := db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("批量插入失败: %v", err)
	}

	return mcp.NewToolResultText(fmt.Sprintf("成功插入%d条数据", len(records))), nil
}

// 获取表结构
func getTableSchema(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	table := request.Params.Arguments["table"].(string)
	dbType := request.Params.Arguments["db_type"].(string)

	var db *sql.DB
	switch dbType {
	case "mysql":
		db = mysqlDB
	case "postgres":
		db = postgresDB
	default:
		return nil, errors.New("不支持的数据库类型")
	}

	if db == nil {
		return nil, errors.New("数据库未连接")
	}

	var query string
	if dbType == "mysql" {
		query = fmt.Sprintf("DESCRIBE %s", table)
	} else {
		query = fmt.Sprintf("SELECT column_name, data_type, is_nullable, column_default FROM information_schema.columns WHERE table_name = '%s'", table)
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询表结构失败: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("获取列名失败: %v", err)
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
			return nil, fmt.Errorf("扫描行失败: %v", err)
		}

		result := make(map[string]interface{})
		for i, col := range columns {
			result[col] = values[i]
		}
		results = append(results, result)
	}

	resultData, err := json.Marshal(results)
	if err != nil {
		return nil, fmt.Errorf("JSON编码失败: %v", err)
	}

	return mcp.NewToolResultText(string(resultData)), nil
}

// 关闭连接
func closeConnection(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dbType := request.Params.Arguments["db_type"].(string)

	switch dbType {
	case "mysql":
		if mysqlDB != nil {
			err := mysqlDB.Close()
			mysqlDB = nil
			if err != nil {
				return nil, fmt.Errorf("关闭MySQL连接失败: %v", err)
			}
			return mcp.NewToolResultText("MySQL连接已关闭"), nil
		}
	case "postgres":
		if postgresDB != nil {
			err := postgresDB.Close()
			postgresDB = nil
			if err != nil {
				return nil, fmt.Errorf("关闭PostgreSQL连接失败: %v", err)
			}
			return mcp.NewToolResultText("PostgreSQL连接已关闭"), nil
		}
	default:
		return nil, errors.New("不支持的数据库类型")
	}

	return mcp.NewToolResultText(fmt.Sprintf("%s连接已关闭", dbType)), nil
}
