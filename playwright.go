package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/playwright-community/playwright-go"
)

// 浏览器实例管理
var browserContext playwright.BrowserContext
var page playwright.Page

// 初始化Playwright
func initPlaywright() error {
	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("启动Playwright失败: %v", err)
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
	})
	if err != nil {
		return fmt.Errorf("启动浏览器失败: %v", err)
	}

	browserContext, err = browser.NewContext()
	if err != nil {
		return fmt.Errorf("创建浏览器上下文失败: %v", err)
	}

	page, err = browserContext.NewPage()
	if err != nil {
		return fmt.Errorf("创建新页面失败: %v", err)
	}

	return nil
}

// 添加浏览器工具函数
func addBrowserTools(mcpServer *server.MCPServer) {
	// 初始化浏览器
	mcpServer.AddTool(mcp.NewTool("browser_init",
		mcp.WithDescription("初始化浏览器")),
		initBrowser)

	// 导航到URL
	mcpServer.AddTool(mcp.NewTool("browser_goto",
		mcp.WithDescription("导航到指定URL"),
		mcp.WithString("url", mcp.Required(), mcp.Description("要导航到的URL"))),
		gotoURL)

	// 获取页面标题
	mcpServer.AddTool(mcp.NewTool("browser_title",
		mcp.WithDescription("获取当前页面标题")),
		getPageTitle)

	// 截图
	mcpServer.AddTool(mcp.NewTool("browser_screenshot",
		mcp.WithDescription("截取当前页面截图"),
		mcp.WithString("path", mcp.Description("截图保存路径"), mcp.DefaultString("screenshot.png"))),
		takeScreenshot)

	// 关闭浏览器
	mcpServer.AddTool(mcp.NewTool("browser_close",
		mcp.WithDescription("关闭浏览器")),
		closeBrowser)
}

// 初始化浏览器
func initBrowser(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	err := initPlaywright()
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText("浏览器初始化成功"), nil
}

// 导航到URL
func gotoURL(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["url"] == nil {
		return nil, errors.New("url参数不能为空")
	}

	url := request.Params.Arguments["url"].(string)
	_, err := page.Goto(url)
	if err != nil {
		return nil, fmt.Errorf("导航到URL失败: %v", err)
	}

	return mcp.NewToolResultText(fmt.Sprintf("已导航到: %s", url)), nil
}

// 获取页面标题
func getPageTitle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title, err := page.Title()
	if err != nil {
		return nil, fmt.Errorf("获取页面标题失败: %v", err)
	}

	return mcp.NewToolResultText(title), nil
}

// 截图
func takeScreenshot(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := "screenshot.png"
	if request.Params.Arguments["path"] != nil {
		path = request.Params.Arguments["path"].(string)
	}

	// 确保目录存在
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return nil, fmt.Errorf("创建目录失败: %v", err)
		}
	}

	_, err := page.Screenshot(playwright.PageScreenshotOptions{
		Path: playwright.String(path),
	})
	if err != nil {
		return nil, fmt.Errorf("截图失败: %v", err)
	}

	absPath, _ := filepath.Abs(path)
	return mcp.NewToolResultText(fmt.Sprintf("截图已保存到: %s", absPath)), nil
}

// 关闭浏览器
func closeBrowser(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if browserContext != nil {
		err := browserContext.Close()
		if err != nil {
			return nil, fmt.Errorf("关闭浏览器失败: %v", err)
		}
		browserContext = nil
		page = nil
	}

	return mcp.NewToolResultText("浏览器已关闭"), nil
}
