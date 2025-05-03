package main

import (
	"context"
	"encoding/json"
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

	// 获取页面内容
	mcpServer.AddTool(mcp.NewTool("browser_content",
		mcp.WithDescription("获取当前页面内容")),
		getPageContent)

	// 执行JavaScript脚本
	mcpServer.AddTool(mcp.NewTool("browser_execute_script",
		mcp.WithDescription("在当前页面执行JavaScript脚本"),
		mcp.WithString("script", mcp.Required(), mcp.Description("要执行的JavaScript脚本"))),
		executeScript)

	// 处理弹窗
	mcpServer.AddTool(mcp.NewTool("browser_handle_dialog",
		mcp.WithDescription("处理页面弹窗"),
		mcp.WithString("action", mcp.Required(), mcp.Description("弹窗操作: accept 或 dismiss")),
		mcp.WithString("prompt_text", mcp.Description("输入框内容，如果有的话"))),
		handleDialog)

	// 点击元素
	mcpServer.AddTool(mcp.NewTool("browser_click_element",
		mcp.WithDescription("点击指定的选择器元素"),
		mcp.WithString("selector", mcp.Required(), mcp.Description("要点击的元素选择器"))),
		clickElement)

	// 输入文本
	mcpServer.AddTool(mcp.NewTool("browser_input_text",
		mcp.WithDescription("在指定元素中输入文本"),
		mcp.WithString("selector", mcp.Required(), mcp.Description("目标输入框选择器")),
		mcp.WithString("text", mcp.Required(), mcp.Description("要输入的文本"))),
		inputText)

	// 获取元素文本
	mcpServer.AddTool(mcp.NewTool("browser_get_element_text",
		mcp.WithDescription("获取指定元素的文本内容"),
		mcp.WithString("selector", mcp.Required(), mcp.Description("目标元素选择器"))),
		getElementText)

	// 等待导航完成
	mcpServer.AddTool(mcp.NewTool("browser_wait_for_navigation",
		mcp.WithDescription("等待页面导航完成"),
		mcp.WithString("timeout", mcp.Description("超时时间（毫秒）"), mcp.DefaultString("30000"))),
		waitForNavigation)

	// 设置窗口大小
	mcpServer.AddTool(mcp.NewTool("browser_set_viewport_size",
		mcp.WithDescription("设置浏览器视口大小"),
		mcp.WithString("width", mcp.Required(), mcp.Description("视口宽度")),
		mcp.WithString("height", mcp.Required(), mcp.Description("视口高度"))),
		setViewportSize)

	// 等待元素可见
	mcpServer.AddTool(mcp.NewTool("browser_wait_for_selector",
		mcp.WithDescription("等待元素可见"),
		mcp.WithString("selector", mcp.Required(), mcp.Description("要等待可见的元素选择器")),
		mcp.WithString("timeout", mcp.Description("超时时间（毫秒）"), mcp.DefaultString("30000"))),
		waitForSelector)

	// 切换标签页
	mcpServer.AddTool(mcp.NewTool("browser_switch_page",
		mcp.WithDescription("切换到指定URL的标签页"),
		mcp.WithString("page_url", mcp.Required(), mcp.Description("要切换到的页面URL"))),
		switchPage)

	// 关闭当前标签页
	mcpServer.AddTool(mcp.NewTool("browser_close_page",
		mcp.WithDescription("关闭当前标签页")),
		closePage)

	// 获取所有Cookies
	mcpServer.AddTool(mcp.NewTool("browser_get_cookies",
		mcp.WithDescription("获取当前浏览器上下文的所有Cookies")),
		getCookies)

	// 设置Cookie
	mcpServer.AddTool(mcp.NewTool("browser_set_cookie",
		mcp.WithDescription("设置Cookie（name=value）"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Cookie名称")),
		mcp.WithString("value", mcp.Required(), mcp.Description("Cookie值"))),
		setCookie)

	// 文件上传处理
	mcpServer.AddTool(mcp.NewTool("browser_upload_file",
		mcp.WithDescription("上传文件到指定元素"),
		mcp.WithString("selector", mcp.Required(), mcp.Description("文件输入框选择器")),
		mcp.WithString("path", mcp.Required(), mcp.Description("要上传的文件路径"))),
		uploadFile)
}

// 点击元素
func clickElement(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["selector"] == nil {
		return nil, errors.New("selector参数不能为空")
	}

	selector := request.Params.Arguments["selector"].(string)
	err := page.Locator(selector).Click()
	if err != nil {
		return nil, fmt.Errorf("点击元素失败: %v", err)
	}

	return mcp.NewToolResultText(fmt.Sprintf("已点击元素: %s", selector)), nil
}

// 输入文本
func inputText(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["selector"] == nil || request.Params.Arguments["text"] == nil {
		return nil, errors.New("selector和text参数都不能为空")
	}

	selector := request.Params.Arguments["selector"].(string)
	text := request.Params.Arguments["text"].(string)
	err := page.Locator(selector).Fill(text)
	if err != nil {
		return nil, fmt.Errorf("输入文本失败: %v", err)
	}

	return mcp.NewToolResultText(fmt.Sprintf("在%s中输入了文本: %s", selector, text)), nil
}

// 获取元素文本
func getElementText(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["selector"] == nil {
		return nil, errors.New("selector参数不能为空")
	}

	selector := request.Params.Arguments["selector"].(string)
	text, err := page.TextContent(selector)
	if err != nil {
		return nil, fmt.Errorf("获取元素文本失败: %v", err)
	}

	return mcp.NewToolResultText(text), nil
}

// 等待导航完成
func waitForNavigation(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	timeout := 30000
	if request.Params.Arguments["timeout"] != nil {
		var ok bool
		timeout, ok = request.Params.Arguments["timeout"].(int)
		if !ok {
			return nil, errors.New("timeout参数必须为整数")
		}
	}

	err := page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		Timeout: playwright.Float(float64(timeout)),
	})
	if err != nil {
		return nil, fmt.Errorf("等待导航完成失败: %v", err)
	}

	return mcp.NewToolResultText("页面导航已完成"), nil
}

// 设置窗口大小
func setViewportSize(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["width"] == nil || request.Params.Arguments["height"] == nil {
		return nil, errors.New("width和height参数都不能为空")
	}

	width, okW := request.Params.Arguments["width"].(int)
	height, okH := request.Params.Arguments["height"].(int)
	if !okW || !okH {
		return nil, errors.New("width和height参数必须为整数")
	}

	err := page.SetViewportSize(width, height)
	if err != nil {
		return nil, fmt.Errorf("设置视口大小失败: %v", err)
	}

	return mcp.NewToolResultText(fmt.Sprintf("视口大小已设置为: %d x %d", width, height)), nil
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

// 获取页面内容
func getPageContent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content, err := page.Content()
	if err != nil {
		return nil, fmt.Errorf("获取页面内容失败: %v", err)
	}
	return mcp.NewToolResultText(content), nil
}

// 执行JavaScript脚本
func executeScript(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["script"] == nil {
		return nil, errors.New("script参数不能为空")
	}

	script := request.Params.Arguments["script"].(string)
	result, err := page.Evaluate(script)
	if err != nil {
		return nil, fmt.Errorf("执行JavaScript脚本失败: %v", err)
	}

	return mcp.NewToolResultText(fmt.Sprintf("脚本执行结果: %v", result)), nil
}

// 处理弹窗
func handleDialog(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["action"] == nil {
		return nil, errors.New("action参数不能为空")
	}

	action := request.Params.Arguments["action"].(string)
	var promptText string
	if request.Params.Arguments["prompt_text"] != nil {
		promptText = request.Params.Arguments["prompt_text"].(string)
	}

	// 监听弹窗事件
	page.On("dialog", func(dialog playwright.Dialog) {
		switch action {
		case "accept":
			dialog.Accept(promptText)
		case "dismiss":
			dialog.Dismiss()
		default:
			dialog.Dismiss()
		}
	})

	return mcp.NewToolResultText("弹窗处理完成"), nil
}

// 等待元素可见
func waitForSelector(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["selector"] == nil {
		return nil, errors.New("selector参数不能为空")
	}

	selector := request.Params.Arguments["selector"].(string)
	timeout := 30000
	if request.Params.Arguments["timeout"] != nil {
		var ok bool
		timeout, ok = request.Params.Arguments["timeout"].(int)
		if !ok {
			return nil, errors.New("timeout参数必须为整数")
		}
	}

	timeoutfloat := float64(timeout)

	err := page.Locator(selector).WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: &timeoutfloat,
	})
	if err != nil {
		return nil, fmt.Errorf("等待元素可见失败: %v", err)
	}

	return mcp.NewToolResultText(fmt.Sprintf("元素 %s 已可见", selector)), nil
}

// 切换标签页
func switchPage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["page_url"] == nil {
		return nil, errors.New("page_url参数不能为空")
	}

	url := request.Params.Arguments["page_url"].(string)
	var newPage playwright.Page
	for _, p := range browserContext.Pages() {
		currentURL := p.URL()
		if currentURL == url {
			newPage = p
			break
		}
	}

	if newPage == nil {
		return nil, fmt.Errorf("未找到指定URL的页面: %s", url)
	}

	page = newPage
	return mcp.NewToolResultText(fmt.Sprintf("已切换到页面: %s", url)), nil
}

// 关闭当前标签页
func closePage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if page == nil {
		return nil, errors.New("当前没有打开的页面")
	}

	err := page.Close()
	if err != nil {
		return nil, fmt.Errorf("关闭页面失败: %v", err)
	}

	// 更新全局页面实例
	pages := browserContext.Pages()
	if len(pages) > 0 {
		page = pages[0]
	} else {
		page = nil
	}

	return mcp.NewToolResultText("当前标签页已关闭"), nil
}

// 获取所有Cookies
func getCookies(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cookies, err := browserContext.Cookies()
	if err != nil {
		return nil, fmt.Errorf("获取Cookies失败: %v", err)
	}

	cookieJSON, _ := json.Marshal(cookies)
	return mcp.NewToolResultText(string(cookieJSON)), nil
}

// 设置Cookie
func setCookie(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["name"] == nil || request.Params.Arguments["value"] == nil {
		return nil, errors.New("name和value参数都不能为空")
	}

	name := request.Params.Arguments["name"].(string)
	value := request.Params.Arguments["value"].(string)

	cookie := playwright.OptionalCookie{
		Name:  name,
		Value: value,
	}

	err := browserContext.AddCookies([]playwright.OptionalCookie{cookie})
	if err != nil {
		return nil, fmt.Errorf("设置Cookie失败: %v", err)
	}

	return mcp.NewToolResultText(fmt.Sprintf("已设置Cookie: %s=%s", name, value)), nil
}

// 文件上传处理
func uploadFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["selector"] == nil || request.Params.Arguments["path"] == nil {
		return nil, errors.New("selector和path参数都不能为空")
	}

	selector := request.Params.Arguments["selector"].(string)
	path := request.Params.Arguments["path"].(string)

	err := page.Locator(selector).SetInputFiles(path)
	if err != nil {
		return nil, fmt.Errorf("文件上传失败: %v", err)
	}

	absPath, _ := filepath.Abs(path)
	return mcp.NewToolResultText(fmt.Sprintf("文件已上传: %s", absPath)), nil
}
