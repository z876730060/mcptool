package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	var transport string
	flag.StringVar(&transport, "t", "sse", "Transport type (stdio or sse)")
	flag.StringVar(&transport, "transport", "sse", "Transport type (stdio or sse)")
	flag.Parse()

	hooks := &server.Hooks{}

	hooks.AddBeforeAny(func(ctx context.Context, id any, method mcp.MCPMethod, message any) {
		fmt.Printf("beforeAny: %s, %v, %v\n", method, id, message)
	})
	hooks.AddOnSuccess(func(ctx context.Context, id any, method mcp.MCPMethod, message any, result any) {
		fmt.Printf("onSuccess: %s, %v, %v, %v\n", method, id, message, result)
	})
	hooks.AddOnError(func(ctx context.Context, id any, method mcp.MCPMethod, message any, err error) {
		fmt.Printf("onError: %s, %v, %v, %v\n", method, id, message, err)
	})
	hooks.AddBeforeInitialize(func(ctx context.Context, id any, message *mcp.InitializeRequest) {
		fmt.Printf("beforeInitialize: %v, %v\n", id, message)
	})
	hooks.AddAfterInitialize(func(ctx context.Context, id any, message *mcp.InitializeRequest, result *mcp.InitializeResult) {
		fmt.Printf("afterInitialize: %v, %v, %v\n", id, message, result)
	})
	hooks.AddAfterCallTool(func(ctx context.Context, id any, message *mcp.CallToolRequest, result *mcp.CallToolResult) {
		fmt.Printf("afterCallTool: %v, %v, %v\n", id, message, result)
	})
	hooks.AddBeforeCallTool(func(ctx context.Context, id any, message *mcp.CallToolRequest) {
		fmt.Printf("beforeCallTool: %v, %v\n", id, message)
	})

	fmt.Println("this is a webserver's mcp-server")
	mcpServer := server.NewMCPServer("windows-assistant", "0.0.0",
		server.WithLogging(),
		server.WithInstructions("支持代码开发工具，GUI操作工具"),
		//server.WithHooks(hooks),
	)

	addCodeTools(mcpServer)
	addComputerTools(mcpServer)
	addCommandTools(mcpServer)
	addBrowserTools(mcpServer)
	addDatabaseTools(mcpServer)
	addToolGenerator(mcpServer)
	registerMemoryTools(mcpServer)
	registerSQLiteTools(mcpServer)
	registerRedisTools(mcpServer)
	addFileBackupTool(mcpServer)
	// 添加屏幕录制工具
	addScreenRecorderTool(mcpServer)

	mcpServer.AddPrompt(mcp.NewPrompt("open_windows_app", mcp.WithPromptDescription("打开windows应用程序"),
		mcp.WithArgument("app")),
		func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			app := request.Params.Arguments["app"]
			if app == "" {
				return nil, errors.New("app 参数不能为空")
			}

			slog.Info("app:" + app)

			return mcp.NewGetPromptResult(fmt.Sprintf(`电脑分辨率1920*1080
1.截图,并使用图片大模型分析当前是否在windows桌面;如果不在,则鼠标移动至(1920,0)点击
2.截图,使用图片大模型分析%s图标坐标
4.如果得到的坐标是图标的左上角,则对坐标做微调,确保能够在图标内,鼠标移动
5.截图,并使用图片大模型分析鼠标是否在%[1]s图标上,如果不在重新提供坐标,并且进行鼠标移动调整
6.双击
7.延迟5秒
8.截图,并使用图片大模型分析%[1]s应用是否展示;如果没展示重新循环以上流程
9.展示结束`, app), []mcp.PromptMessage{}), nil
		})

	mcpServer.AddTool(mcp.NewTool("sleep",
		mcp.WithDescription("延迟"),
		mcp.WithString("duration", mcp.Description("延迟时间(example:5s)"))),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

			if nil == request.Params.Arguments["duration"] {
				return nil, errors.New("duration 参数不能为空")
			}

			duration, err := time.ParseDuration(request.Params.Arguments["duration"].(string))
			if err != nil {
				return nil, err
			}

			time.Sleep(duration)
			return mcp.NewToolResultText("延迟结束"), nil
		})

	mcpServer.AddTool(mcp.NewTool("network_search",
		mcp.WithDescription("Fetches a URL from the internet and extracts its contents as markdown. "),
		mcp.WithString("url", mcp.Required(), mcp.Description("URL to fetch")),
		mcp.WithNumber("max_length", mcp.Description("Maximum number of characters to return (default: 5000)")),
		mcp.WithNumber("start_index", mcp.Description("Start content from this character index (default: 0)")),
		mcp.WithBoolean("raw", mcp.Description("Get raw content without markdown conversion (default: false)"))),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			fmt.Println(request.Params.Arguments)
			url := request.Params.Arguments["url"].(string)
			resp, err := http.Get(url)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			all, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(all)), nil
		})

	if transport == "sse" {
		sseServer := server.NewSSEServer(mcpServer, server.WithBaseURL("http://localhost:8081"))
		log.Printf("SSE server listening on :8081")
		if err := sseServer.Start(":8081"); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	} else {
		if err := server.ServeStdio(mcpServer); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}
