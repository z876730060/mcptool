package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// FetchManager 管理HTTP获取操作的结构体
type FetchManager struct {
	userAgent    string
	ignoreRobots bool
	proxyURL     string
	client       *http.Client
}

// fetchHTTP 发起HTTP请求并返回内容和类型
func (fm *FetchManager) fetchHTTP(url string) (string, string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", err
	}

	if fm.userAgent != "" {
		req.Header.Set("User-Agent", fm.userAgent)
	} else {
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MCP-Fetch/1.0)")
	}

	resp, err := fm.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("HTTP请求失败，状态码: %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	return string(content), resp.Header.Get("Content-Type"), nil
}

// checkRobotsTxt 检查robots.txt规则
func (fm *FetchManager) checkRobotsTxt(url1 string) (bool, error) {
	parsedURL, err := url.Parse(url1)
	if err != nil {
		return false, err
	}

	robotsURL := fmt.Sprintf("%s://%s/robots.txt", parsedURL.Scheme, parsedURL.Host)
	_, _, err = fm.fetchHTTP(robotsURL)
	// 简化处理，实际应解析robots.txt内容
	return true, nil
}

// htmlToMarkdown 将HTML转换为Markdown
func (fm *FetchManager) htmlToMarkdown(html string) (string, error) {
	// 使用github.com/JohannesKaufmann/html-to-markdown库实现HTML转Markdown
	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(html)
	if err != nil {
		return "", fmt.Errorf("HTML转Markdown失败: %v", err)
	}
	return markdown, nil
}

// NewFetchManager 创建新的FetchManager实例
func NewFetchManager(userAgent string, ignoreRobots bool, proxyURL string) (*FetchManager, error) {
	log.Printf("正在初始化FetchManager: userAgent=%s, ignoreRobots=%v, proxyURL=%s", userAgent, ignoreRobots, proxyURL)

	transport := &http.Transport{
		ResponseHeaderTimeout: 30 * time.Second,
	}

	if proxyURL != "" {
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("无效的代理URL: %v", err)
		}
		transport.Proxy = http.ProxyURL(proxy)
	}

	return &FetchManager{
		userAgent:    userAgent,
		ignoreRobots: ignoreRobots,
		proxyURL:     proxyURL,
		client:       &http.Client{Transport: transport, Timeout: 60 * time.Second},
	}, nil
}

// FetchURL 获取URL内容
func (fm *FetchManager) FetchURL(url string, forceRaw bool) (string, string, error) {
	log.Printf("正在获取URL内容: %s", url)

	// 1. 检查robots.txt
	if !fm.ignoreRobots {
		allowed, err := fm.checkRobotsTxt(url)
		if err != nil {
			return "", "", fmt.Errorf("检查robots.txt失败: %v", err)
		}
		if !allowed {
			return "", "", fmt.Errorf("根据robots.txt规则，不允许抓取此URL")
		}
	}

	// 2. 发起HTTP请求
	content, contentType, err := fm.fetchHTTP(url)
	if err != nil {
		return "", "", fmt.Errorf("HTTP请求失败: %v", err)
	}

	// 3. 根据forceRaw决定是否转换
	var result string
	if forceRaw || !strings.Contains(contentType, "text/html") {
		result = content
	} else {
		// 4. HTML转Markdown
		result, err = fm.htmlToMarkdown(content)
		if err != nil {
			return "", "", fmt.Errorf("HTML转Markdown失败: %v", err)
		}
	}

	// 5. 返回结果
	prefix := fmt.Sprintf("内容来自: %s\n\n", url)
	return result, prefix, nil
}

// 注册Fetch工具
func registerFetchTools(mcpServer *server.MCPServer) {
	mcpServer.AddTool(mcp.NewTool("fetch",
		mcp.WithDescription("从互联网获取URL内容"),
		mcp.WithString("url", mcp.Required(), mcp.Description("要获取的URL"))),
		fetchHandler)
}

// fetchHandler 处理fetch工具调用
func fetchHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url := request.Params.Arguments["url"].(string)
	fm, err := NewFetchManager("", false, "")
	if err != nil {
		return nil, err
	}
	content, prefix, err := fm.FetchURL(url, true)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(fmt.Sprintf("%s%s", prefix, content)), nil
}
