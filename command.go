package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// 命令历史记录
var commandHistory []map[string]interface{}

// 最大历史记录数量
const maxHistorySize = 50

func addCommandTools(mcpServer *server.MCPServer) {
	// 添加终端命令执行工具
	mcpServer.AddTool(mcp.NewTool("command_execute",
		mcp.WithDescription("执行终端命令"),
		mcp.WithString("command", mcp.Required(), mcp.Description("要执行的命令")),
		mcp.WithNumber("timeout", mcp.Description("命令超时时间(秒),默认30秒"), mcp.DefaultNumber(30))),
		executeCommand)

	// 添加获取命令历史工具
	mcpServer.AddTool(mcp.NewTool("command_history",
		mcp.WithDescription("获取命令执行历史"),
		mcp.WithNumber("count", mcp.Description("要获取的历史记录数量,默认10秒"), mcp.DefaultNumber(10))),
		getCommandHistory)

	// 添加获取当前目录工具
	mcpServer.AddTool(mcp.NewTool("command_current_dir",
		mcp.WithDescription("获取当前工作目录")),
		getCurrentDirectory)

	// 添加切换目录工具
	mcpServer.AddTool(mcp.NewTool("command_change_dir",
		mcp.WithDescription("切换工作目录"),
		mcp.WithString("path", mcp.Required(), mcp.Description("要切换到的目录路径"))),
		changeDirectory)
}

func executeCommand(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["command"] == nil {
		return nil, errors.New("command 参数不能为空")
	}

	command := request.Params.Arguments["command"].(string)
	timeout := 30.0
	if request.Params.Arguments["timeout"] != nil {
		timeout = request.Params.Arguments["timeout"].(float64)
	}

	// 检查危险命令
	dangerousCommands := []string{"rm -rf /", "mkfs"}
	for _, dc := range dangerousCommands {
		if strings.Contains(strings.ToLower(command), dc) {
			return mcp.NewToolResultText("出于安全考虑，不允许执行此命令"), nil
		}
	}

	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// 执行命令
	cmd := exec.CommandContext(ctx, "cmd", "/C", command)
	if runtime.GOOS != "windows" {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	// 添加到历史记录
	commandHistory = append(commandHistory, map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"command":   command,
		"success":   err == nil,
	})

	// 限制历史记录大小
	if len(commandHistory) > maxHistorySize {
		commandHistory = commandHistory[1:]
	}

	if runtime.GOOS == "windows" {
		reader := transform.NewReader(bytes.NewReader(output), simplifiedchinese.GBK.NewDecoder())
		output, err = io.ReadAll(reader)
	}

	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("命令执行失败(耗时: %v)\n错误: %v\n输出: %s",
			duration, err, string(output))), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("命令执行成功(耗时: %v)\n输出: %s",
		duration, string(output))), nil
}

func getCommandHistory(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	count := 10
	if request.Params.Arguments["count"] != nil {
		count = int(request.Params.Arguments["count"].(float64))
	}

	if len(commandHistory) == 0 {
		return mcp.NewToolResultText("没有命令执行历史"), nil
	}

	count = min(count, len(commandHistory))
	recentCommands := commandHistory[len(commandHistory)-count:]

	output := fmt.Sprintf("最近 %d 条命令历史:\n\n", count)
	for i, cmd := range recentCommands {
		status := "✓"
		if !cmd["success"].(bool) {
			status = "✗"
		}
		output += fmt.Sprintf("%d. [%s] %s: %s\n",
			i+1, status, cmd["timestamp"].(string), cmd["command"].(string))
	}

	return mcp.NewToolResultText(output), nil
}

func getCurrentDirectory(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(currentDir), nil
}

func changeDirectory(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["path"] == nil {
		return nil, errors.New("path 参数不能为空")
	}

	path := request.Params.Arguments["path"].(string)
	err := os.Chdir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return mcp.NewToolResultText(fmt.Sprintf("错误: 目录 '%s' 不存在", path)), nil
		}
		return nil, err
	}

	currentDir, _ := os.Getwd()
	return mcp.NewToolResultText(fmt.Sprintf("已切换到目录: %s", currentDir)), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
