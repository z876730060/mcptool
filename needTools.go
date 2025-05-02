package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// 添加工具生成器功能
func addToolGenerator(mcpServer *server.MCPServer) {
	mcpServer.AddTool(mcp.NewTool("generate_tool",
		mcp.WithDescription("在用户提出的需求基础上，大模型判断现有的工具无法满足要求，需要创建新的工具，就需要调用这个工具生成新的工具描述文件。"),
		mcp.WithString("tool_key", mcp.Required(), mcp.Description("工具唯一标识key")),
		mcp.WithString("purpose", mcp.Required(), mcp.Description("工具目的")),
		mcp.WithString("functionality", mcp.Required(), mcp.Description("工具功能(特别详细的描述,包括输入输出,以及可能的异常情况)")),
		mcp.WithString("parameters", mcp.Required(), mcp.Description("工具参数")),
		mcp.WithString("expected_result", mcp.Required(), mcp.Description("预期结果"))),
		generateToolDescription)

	mcpServer.AddTool(mcp.NewTool("improve_tool",
		mcp.WithDescription("当现有工具无法满足需求时，提出改进方案并生成改进文件"),
		mcp.WithString("tool_key", mcp.Required(), mcp.Description("需要改进的工具唯一标识key")),
		mcp.WithString("problem_description", mcp.Required(), mcp.Description("当前遇到的问题详细描述")),
		mcp.WithString("current_limitations", mcp.Required(), mcp.Description("现有工具的局限性分析")),
		mcp.WithString("improvement_suggestions", mcp.Required(), mcp.Description("具体的改进建议")),
		mcp.WithString("implementation_steps", mcp.Required(), mcp.Description("实施步骤"))),
		generateToolImprovement)
}

// 生成工具描述文件
func generateToolDescription(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	toolKey := request.Params.Arguments["tool_key"].(string)
	purpose := request.Params.Arguments["purpose"].(string)
	functionality := request.Params.Arguments["functionality"].(string)
	parameters := request.Params.Arguments["parameters"].(string)
	expectedResult := request.Params.Arguments["expected_result"].(string)

	// 创建tools目录如果不存在
	if _, err := os.Stat("tools"); os.IsNotExist(err) {
		err = os.Mkdir("tools", 0755)
		if err != nil {
			return nil, fmt.Errorf("创建tools目录失败: %v", err)
		}
	}

	// 生成工具描述文件
	fileName := filepath.Join("tools", toolKey+".txt")
	content := fmt.Sprintf("工具目的: %s\n工具功能: %s\n工具参数: %s\n预期结果: %s",
		purpose, functionality, parameters, expectedResult)

	err := os.WriteFile(fileName, []byte(content), 0644)
	if err != nil {
		return nil, fmt.Errorf("写入工具描述文件失败: %v", err)
	}

	absPath, _ := filepath.Abs(fileName)
	return mcp.NewToolResultText(fmt.Sprintf("工具描述文件已生成: %s", absPath)), nil
}

// 生成工具改进方案文件
func generateToolImprovement(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	toolKey := request.Params.Arguments["tool_key"].(string)
	problemDescription := request.Params.Arguments["problem_description"].(string)
	currentLimitations := request.Params.Arguments["current_limitations"].(string)
	improvementSuggestions := request.Params.Arguments["improvement_suggestions"].(string)
	implementationSteps := request.Params.Arguments["implementation_steps"].(string)

	// 创建improvements目录如果不存在
	if _, err := os.Stat("tools/improvements"); os.IsNotExist(err) {
		err = os.Mkdir("tools/improvements", 0755)
		if err != nil {
			return nil, fmt.Errorf("创建improvements目录失败: %v", err)
		}
	}

	// 生成工具改进方案文件
	fileName := filepath.Join("tools/improvements", toolKey+"_improvement.txt")
	content := fmt.Sprintf("问题描述:\n%s\n\n现有工具局限性:\n%s\n\n改进建议:\n%s\n\n实施步骤:\n%s",
		problemDescription, currentLimitations, improvementSuggestions, implementationSteps)

	err := os.WriteFile(fileName, []byte(content), 0644)
	if err != nil {
		return nil, fmt.Errorf("写入工具改进方案文件失败: %v", err)
	}

	absPath, _ := filepath.Abs(fileName)
	return mcp.NewToolResultText(fmt.Sprintf("工具改进方案文件已生成: %s", absPath)), nil
}
