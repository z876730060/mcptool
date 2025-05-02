package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// FileBackupResult 备份结果结构体
type FileBackupResult struct {
	SuccessCount int      `json:"success_count"`
	FailedFiles  []string `json:"failed_files"`
	Errors       []string `json:"errors"`
}

// backupFiles 执行文件备份
func backupFiles(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourcePath := request.Params.Arguments["source_path"].(string)
	targetPath := request.Params.Arguments["target_path"].(string)
	overwrite := request.Params.Arguments["overwrite"].(bool)

	result := &FileBackupResult{}

	// 检查源目录是否存在
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("源目录不存在: %s", sourcePath)
	}

	// 创建目标目录如果不存在
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		err = os.MkdirAll(targetPath, 0755)
		if err != nil {
			return nil, fmt.Errorf("创建目标目录失败: %v", err)
		}
	}

	// 遍历源目录并复制文件
	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("访问文件错误: %s - %v", path, err))
			return nil
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("获取相对路径错误: %s - %v", path, err))
			return nil
		}

		destPath := filepath.Join(targetPath, relPath)

		// 检查目标文件是否存在
		if _, err := os.Stat(destPath); err == nil && !overwrite {
			result.Errors = append(result.Errors, fmt.Sprintf("跳过已存在文件: %s", destPath))
			return nil
		}

		// 创建目标文件的目录结构
		destDir := filepath.Dir(destPath)
		if _, err := os.Stat(destDir); os.IsNotExist(err) {
			err = os.MkdirAll(destDir, 0755)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("创建目录失败: %s - %v", destDir, err))
				return nil
			}
		}

		// 复制文件
		err = copyFile(path, destPath)
		if err != nil {
			result.FailedFiles = append(result.FailedFiles, path)
			result.Errors = append(result.Errors, fmt.Sprintf("复制文件失败: %s - %v", path, err))
			log.Printf("文件备份失败: %s, 错误: %v", path, err)
		} else {
			result.SuccessCount++
			log.Printf("成功备份文件: %s -> %s", path, destPath)
			log.Printf("备份进度: 成功 %d, 失败 %d", result.SuccessCount, len(result.FailedFiles))
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("遍历源目录错误: %v", err)
	}

	return mcp.NewToolResultText(EncodeSerializedData(result)), nil
}

// copyFile 复制单个文件
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer source.Close()

	destination, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer destination.Close()

	// 使用带缓冲的复制提高性能
	_, err = io.CopyBuffer(destination, source, make([]byte, 32*1024)) // 32KB缓冲区
	if err != nil {
		return fmt.Errorf("文件复制失败: %w", err)
	}

	// 确保所有数据写入磁盘
	err = destination.Sync()
	if err != nil {
		return fmt.Errorf("同步文件失败: %w", err)
	}

	return nil
}

// 添加文件备份工具
func addFileBackupTool(mcpServer *server.MCPServer) {
	mcpServer.AddTool(mcp.NewTool("file_backup",
		mcp.WithDescription("备份指定目录下的文件到目标路径"),
		mcp.WithString("source_path", mcp.Required(), mcp.Description("源目录路径")),
		mcp.WithString("target_path", mcp.Required(), mcp.Description("目标备份路径")),
		mcp.WithBoolean("overwrite", mcp.DefaultBool(false), mcp.Description("是否覆盖已有文件"))),
		backupFiles)
}
