package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// GitManager 管理Git操作的结构体
type GitManager struct {
	repo *git.Repository
}

// NewGitManager 创建新的GitManager实例
func NewGitManager(repoPath string) (*GitManager, error) {
	log.Printf("正在打开Git仓库: %s", repoPath)
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		log.Printf("打开Git仓库失败: %v", err)
		return nil, fmt.Errorf("打开Git仓库失败: %v", err)
	}
	log.Printf("成功打开Git仓库: %s", repoPath)
	return &GitManager{repo: repo}, nil
}

// Status 获取仓库状态
func (gm *GitManager) Status() (string, error) {
	log.Println("正在获取仓库状态")
	worktree, err := gm.repo.Worktree()
	if err != nil {
		log.Printf("获取工作树失败: %v", err)
		return "", err
	}
	status, err := worktree.Status()
	if err != nil {
		log.Printf("获取状态失败: %v", err)
		return "", err
	}
	log.Println("成功获取仓库状态")
	return status.String(), nil
}

// DiffUnstaged 获取未暂存的变更
func (gm *GitManager) DiffUnstaged() (string, error) {
	log.Println("正在获取未暂存的变更")
	worktree, err := gm.repo.Worktree()
	if err != nil {
		log.Printf("获取工作树失败: %v", err)
		return "", err
	}
	status, err := worktree.Status()
	if err != nil {
		log.Printf("获取状态失败: %v", err)
		return "", err
	}
	var sb strings.Builder
	for file, s := range status {
		if s.Worktree != git.Unmodified {
			sb.WriteString(fmt.Sprintf("%s: %s\n", file, s.Worktree))
		}
	}
	log.Println("成功获取未暂存的变更")
	return sb.String(), nil
}

// DiffStaged 获取已暂存的变更
func (gm *GitManager) DiffStaged() (string, error) {
	log.Println("正在获取已暂存的变更")
	worktree, err := gm.repo.Worktree()
	if err != nil {
		log.Printf("获取工作树失败: %v", err)
		return "", err
	}
	status, err := worktree.Status()
	if err != nil {
		log.Printf("获取状态失败: %v", err)
		return "", err
	}
	var sb strings.Builder
	for file, s := range status {
		if s.Staging != git.Unmodified {
			sb.WriteString(fmt.Sprintf("%s: %s\n", file, s.Staging))
		}
	}
	log.Println("成功获取已暂存的变更")
	return sb.String(), nil
}

// 注册Git工具
func registerGitTools(mcpServer *server.MCPServer) {
	mcpServer.AddTool(mcp.NewTool("git_status",
		mcp.WithDescription("显示工作树状态"),
		mcp.WithString("repo_path", mcp.Required(), mcp.Description("Git仓库路径"))),
		gitStatusHandler)

	mcpServer.AddTool(mcp.NewTool("git_diff_unstaged",
		mcp.WithDescription("显示工作目录中未暂存的变更"),
		mcp.WithString("repo_path", mcp.Required(), mcp.Description("Git仓库路径"))),
		gitDiffUnstagedHandler)

	mcpServer.AddTool(mcp.NewTool("git_diff_staged",
		mcp.WithDescription("显示已暂存准备提交的变更"),
		mcp.WithString("repo_path", mcp.Required(), mcp.Description("Git仓库路径"))),
		gitDiffStagedHandler)

	// 可以继续添加其他Git工具...

	mcpServer.AddTool(mcp.NewTool("git_commit",
		mcp.WithDescription("提交暂存的变更"),
		mcp.WithString("repo_path", mcp.Required(), mcp.Description("Git仓库路径")),
		mcp.WithString("message", mcp.Required(), mcp.Description("提交信息"))),
		gitCommitHandler)

	mcpServer.AddTool(mcp.NewTool("git_branch",
		mcp.WithDescription("列出所有分支"),
		mcp.WithString("repo_path", mcp.Required(), mcp.Description("Git仓库路径"))),
		gitBranchHandler)

	mcpServer.AddTool(mcp.NewTool("git_log",
		mcp.WithDescription("显示提交日志"),
		mcp.WithString("repo_path", mcp.Required(), mcp.Description("Git仓库路径")),
		mcp.WithNumber("limit", mcp.DefaultNumber(10), mcp.Description("日志条目限制"))),
		gitLogHandler)
}

// gitStatusHandler 处理git_status工具调用
func gitStatusHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoPath := request.Params.Arguments["repo_path"].(string)
	gm, err := NewGitManager(repoPath)
	if err != nil {
		return nil, err
	}
	status, err := gm.Status()
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(fmt.Sprintf("仓库状态:\n%s", status)), nil
}

// gitDiffUnstagedHandler 处理git_diff_unstaged工具调用
func gitDiffUnstagedHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoPath := request.Params.Arguments["repo_path"].(string)
	gm, err := NewGitManager(repoPath)
	if err != nil {
		return nil, err
	}
	diff, err := gm.DiffUnstaged()
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(fmt.Sprintf("未暂存的变更:\n%s", diff)), nil
}

// gitDiffStagedHandler 处理git_diff_staged工具调用
func gitDiffStagedHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoPath := request.Params.Arguments["repo_path"].(string)
	gm, err := NewGitManager(repoPath)
	if err != nil {
		return nil, err
	}
	diff, err := gm.DiffStaged()
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(fmt.Sprintf("已暂存的变更:\n%s", diff)), nil
}

// Commit 提交暂存的变更
func (gm *GitManager) Commit(message string) (string, error) {
	log.Printf("正在提交变更，消息: %s", message)
	worktree, err := gm.repo.Worktree()
	if err != nil {
		log.Printf("获取工作树失败: %v", err)
		return "", err
	}
	commit, err := worktree.Commit(message, &git.CommitOptions{})
	if err != nil {
		log.Printf("提交失败: %v", err)
		return "", err
	}
	log.Printf("提交成功: %s", commit.String())
	return fmt.Sprintf("提交成功: %s", commit.String()), nil
}

// ListBranches 列出所有分支
func (gm *GitManager) ListBranches() (string, error) {
	log.Println("正在列出分支")
	branches, err := gm.repo.Branches()
	if err != nil {
		log.Printf("获取分支列表失败: %v", err)
		return "", err
	}
	var sb strings.Builder
	err = branches.ForEach(func(ref *plumbing.Reference) error {
		sb.WriteString(fmt.Sprintf("%s\n", ref.Name().Short()))
		return nil
	})
	if err != nil {
		log.Printf("遍历分支失败: %v", err)
		return "", err
	}
	log.Println("成功列出分支")
	return sb.String(), nil
}

// GetLog 获取提交日志
func (gm *GitManager) GetLog(limit int) (string, error) {
	log.Printf("正在获取提交日志，限制: %d", limit)
	iter, err := gm.repo.Log(&git.LogOptions{})
	if err != nil {
		log.Printf("获取日志迭代器失败: %v", err)
		return "", err
	}
	var sb strings.Builder
	count := 0
	err = iter.ForEach(func(commit *object.Commit) error {
		if count >= limit {
			return nil
		}
		sb.WriteString(fmt.Sprintf("Commit: %s\nAuthor: %s\nDate: %s\nMessage: %s\n\n",
			commit.Hash.String(), commit.Author.String(), commit.Author.When, commit.Message))
		count++
		return nil
	})
	if err != nil {
		log.Printf("遍历提交日志失败: %v", err)
		return "", err
	}
	log.Printf("成功获取 %d 条提交日志", count)
	return sb.String(), nil
}

// gitCommitHandler 处理git_commit工具调用
func gitCommitHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoPath := request.Params.Arguments["repo_path"].(string)
	message := request.Params.Arguments["message"].(string)
	gm, err := NewGitManager(repoPath)
	if err != nil {
		return nil, err
	}
	result, err := gm.Commit(message)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(result), nil
}

// gitBranchHandler 处理git_branch工具调用
func gitBranchHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoPath := request.Params.Arguments["repo_path"].(string)
	gm, err := NewGitManager(repoPath)
	if err != nil {
		return nil, err
	}
	branches, err := gm.ListBranches()
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(fmt.Sprintf("分支列表:\n%s", branches)), nil
}

// gitLogHandler 处理git_log工具调用
func gitLogHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoPath := request.Params.Arguments["repo_path"].(string)
	limit := request.Params.Arguments["limit"].(int)
	gm, err := NewGitManager(repoPath)
	if err != nil {
		return nil, err
	}
	log, err := gm.GetLog(limit)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(fmt.Sprintf("提交日志:\n%s", log)), nil
}
