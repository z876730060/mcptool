package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"golang.org/x/text/encoding/simplifiedchinese"
)

func addCodeTools(mcpServer *server.MCPServer) {
	mcpServer.AddTool(mcp.NewTool("create_dir", mcp.WithDescription("创建文件夹"),
		mcp.WithString("path", mcp.Required(), mcp.Description("文件夹的路径"))),
		createDir)

	mcpServer.AddTool(mcp.NewTool("read_file", mcp.WithDescription("读取代码文件"),
		mcp.WithString("file", mcp.Required(), mcp.Description("代码文件路径"))),
		readFile)

	mcpServer.AddTool(mcp.NewTool("read_file_offset", mcp.WithDescription("偏移读取文件内容"),
		mcp.WithString("file", mcp.Required(), mcp.Description("代码文件路径")),
		mcp.WithNumber("offset", mcp.Required(), mcp.Description("偏移量")),
		mcp.WithNumber("n", mcp.Required(), mcp.Description("读取字节数"))),
		readFileOffset)

	mcpServer.AddTool(mcp.NewTool("write_file", mcp.WithDescription("写入代码文件"),
		mcp.WithString("file", mcp.Required(), mcp.Description("代码文件路径")),
		mcp.WithString("code", mcp.Required(), mcp.Description("代码文件的全部内容"))),
		writeFile)

	mcpServer.AddTool(mcp.NewTool("write_file_offset", mcp.WithDescription("偏移写入代码文件"),
		mcp.WithString("file", mcp.Required(), mcp.Description("代码文件路径")),
		mcp.WithNumber("offset", mcp.Required(), mcp.Description("偏移量")),
		mcp.WithString("code", mcp.Required(), mcp.Description("代码文件的全部内容"))),
		writeFileOffset)

	mcpServer.AddTool(mcp.NewTool("delete_file", mcp.WithDescription("删除文件"),
		mcp.WithString("file", mcp.Required(), mcp.Description("文件名称或绝对路径"))),
		deleteFile)

	mcpServer.AddTool(mcp.NewTool("delete_dir", mcp.WithDescription("删除目录以及目录下所有文件（谨慎使用）"),
		mcp.WithString("path", mcp.Required(), mcp.Description("目录路径或绝对路径"))),
		deleteDirs)

	// 新增工具函数
	mcpServer.AddTool(mcp.NewTool("list_dir", mcp.WithDescription("列出目录内容"),
		mcp.WithString("path", mcp.Required(), mcp.Description("目录路径"))),
		listDir)

	mcpServer.AddTool(mcp.NewTool("move_file", mcp.WithDescription("移动/重命名文件"),
		mcp.WithString("source", mcp.Required(), mcp.Description("源文件路径")),
		mcp.WithString("destination", mcp.Required(), mcp.Description("目标路径"))),
		moveFile)

	mcpServer.AddTool(mcp.NewTool("search_files", mcp.WithDescription("搜索文件"),
		mcp.WithString("path", mcp.Required(), mcp.Description("搜索根目录")),
		mcp.WithString("pattern", mcp.Required(), mcp.Description("文件名匹配模式")),
		mcp.WithArray("excludePatterns", mcp.Description("排除模式列表"))),
		searchFiles)

	mcpServer.AddTool(mcp.NewTool("get_file_info", mcp.WithDescription("获取文件信息"),
		mcp.WithString("path", mcp.Required(), mcp.Description("文件路径"))),
		getFileInfo)

	mcpServer.AddTool(mcp.NewTool("edit_file", mcp.WithDescription("编辑文件内容"),
		mcp.WithString("path", mcp.Required(), mcp.Description("文件路径")),
		mcp.WithArray("edits", mcp.Required(), mcp.Description("编辑操作列表")),
		mcp.WithBoolean("dryRun", mcp.Description("是否仅预览更改"))),
		editFile)
}

func getOs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	currentOs := os.Getenv("OS")
	return mcp.NewToolResultText(currentOs), nil
}

func currentPath(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	abs, err := filepath.Abs(".")
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(abs), nil
}

func createDir(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["path"] == nil {
		return nil, errors.New("path 参数不能为空")
	}

	path := request.Params.Arguments["path"].(string)
	err := os.MkdirAll(path, os.ModeDir)
	if err != nil {
		if os.IsExist(err) {
			return mcp.NewToolResultText("文件夹已经存在"), nil
		}
		return nil, err
	}
	return mcp.NewToolResultText("创建成功"), nil
}

func readFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["file"] == nil {
		return nil, errors.New("file 参数不能为空")
	}

	file := request.Params.Arguments["file"].(string)
	fileBs, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("代码文件不存在")
		}
		return nil, err
	}
	return mcp.NewToolResultText(string(fileBs)), nil
}

func readFileOffset(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["file"] == nil {
		return nil, errors.New("file 参数不能为空")
	}

	if request.Params.Arguments["offset"] == nil {
		return nil, errors.New("offset 参数不能为空")
	}

	if request.Params.Arguments["n"] == nil {
		return nil, errors.New("n 参数不能为空")
	}

	file := request.Params.Arguments["file"].(string)

	offset, err := strconv.ParseInt(fmt.Sprintf("%.f", request.Params.Arguments["offset"].(float64)), 0, 64)
	if err != nil {
		return nil, err
	}

	n, err := strconv.ParseInt(fmt.Sprintf("%.f", request.Params.Arguments["n"].(float64)), 0, 64)
	if err != nil {
		return nil, err
	}

	openFile, err := os.OpenFile(file, os.O_WRONLY, os.ModeType)
	if err != nil {
		return nil, err
	}
	defer openFile.Close()

	sectionReader := io.NewSectionReader(openFile, offset, n)
	all, err := io.ReadAll(sectionReader)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(string(all)), nil
}

func writeFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["file"] == nil {
		return nil, errors.New("file 参数不能为空")
	}
	if request.Params.Arguments["code"] == nil {
		return nil, errors.New("code 参数不能为空")
	}

	file := request.Params.Arguments["file"].(string)
	code := request.Params.Arguments["code"].(string)

	openFile, err := os.OpenFile(file, os.O_WRONLY, os.ModePerm)
	if err != nil {
		if os.IsNotExist(err) {
			openFile, err = os.Create(file)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	defer openFile.Close()

	_, err = io.WriteString(openFile, code)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText("代码写入成功"), nil
}

func writeFileOffset(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["file"] == nil {
		return nil, errors.New("file 参数不能为空")
	}
	if request.Params.Arguments["offset"] == nil {
		return nil, errors.New("offset 参数不能为空")
	}
	if request.Params.Arguments["code"] == nil {
		return nil, errors.New("code 参数不能为空")
	}

	file := request.Params.Arguments["file"].(string)
	code := request.Params.Arguments["code"].(string)
	offset, err := strconv.ParseInt(fmt.Sprintf("%.f", request.Params.Arguments["offset"].(float64)), 0, 64)
	if err != nil {
		return nil, err
	}

	openFile, err := os.OpenFile(file, os.O_WRONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer openFile.Close()
	writer := io.NewOffsetWriter(openFile, offset)
	_, err = io.WriteString(writer, code)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText("写入成功"), nil
}

// listDir 列出目录内容
func listDir(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["path"] == nil {
		return nil, errors.New("path 参数不能为空")
	}

	path := request.Params.Arguments["path"].(string)
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %v", err)
	}

	var result []string
	for _, entry := range entries {
		result = append(result, entry.Name())
	}

	resultData, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("序列化结果失败: %v", err)
	}

	return mcp.NewToolResultText(string(resultData)), nil
}

// moveFile 移动或重命名文件
func moveFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["source"] == nil {
		return nil, errors.New("source 参数不能为空")
	}
	if request.Params.Arguments["destination"] == nil {
		return nil, errors.New("destination 参数不能为空")
	}

	source := request.Params.Arguments["source"].(string)
	destination := request.Params.Arguments["destination"].(string)

	err := os.Rename(source, destination)
	if err != nil {
		return nil, fmt.Errorf("移动文件失败: %v", err)
	}

	return mcp.NewToolResultText("文件移动成功"), nil
}

// searchFiles 搜索文件
func searchFiles(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["path"] == nil {
		return nil, errors.New("path 参数不能为空")
	}
	if request.Params.Arguments["pattern"] == nil {
		return nil, errors.New("pattern 参数不能为空")
	}

	rootPath := request.Params.Arguments["path"].(string)
	pattern := request.Params.Arguments["pattern"].(string)
	var excludePatterns []string
	if request.Params.Arguments["excludePatterns"] != nil {
		excludePatterns = request.Params.Arguments["excludePatterns"].([]string)
	}

	var results []string
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		matched, err := filepath.Match(pattern, info.Name())
		if err != nil {
			return err
		}

		if matched {
			for _, exclude := range excludePatterns {
				excludeMatched, err := filepath.Match(exclude, info.Name())
				if err != nil {
					return err
				}
				if excludeMatched {
					return nil
				}
			}
			results = append(results, path)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("搜索文件失败: %v", err)
	}

	resultData, err := json.Marshal(results)
	if err != nil {
		return nil, fmt.Errorf("序列化结果失败: %v", err)
	}

	return mcp.NewToolResultText(string(resultData)), nil
}

// getFileInfo 获取文件信息
func getFileInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["path"] == nil {
		return nil, errors.New("path 参数不能为空")
	}

	path := request.Params.Arguments["path"].(string)
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %v", err)
	}

	result := map[string]interface{}{
		"name":      info.Name(),
		"size":      info.Size(),
		"mode":      info.Mode().String(),
		"modTime":   info.ModTime().Format(time.RFC3339),
		"isDir":     info.IsDir(),
		"isRegular": info.Mode().IsRegular(),
	}

	resultData, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("序列化结果失败: %v", err)
	}

	return mcp.NewToolResultText(string(resultData)), nil
}

// editFile 编辑文件内容
// EditOperation 定义文件编辑操作
type EditOperation struct {
	OldText string `json:"oldText"` // 要替换的旧文本
	NewText string `json:"newText"` // 替换后的新文本
}

func editFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["path"] == nil {
		return nil, errors.New("path 参数不能为空")
	}
	if request.Params.Arguments["edits"] == nil {
		return nil, errors.New("edits 参数不能为空")
	}

	path := request.Params.Arguments["path"].(string)
	var edits []EditOperation
	DecodeSerializedData(request.Params.Arguments["edits"], &edits)
	dryRun := false
	if request.Params.Arguments["dryRun"] != nil {
		dryRun = request.Params.Arguments["dryRun"].(bool)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %v", err)
	}

	text := string(content)
	var changes []string

	for _, editOp := range edits {
		oldText := editOp.OldText
		newText := editOp.NewText

		if !strings.Contains(text, oldText) {
			return nil, fmt.Errorf("未找到匹配的文本: %s", oldText)
		}

		changes = append(changes, fmt.Sprintf("替换: %q -> %q", oldText, newText))
		if !dryRun {
			text = strings.ReplaceAll(text, oldText, newText)
		}
	}

	if !dryRun {
		err = os.WriteFile(path, []byte(text), 0644)
		if err != nil {
			return nil, fmt.Errorf("写入文件失败: %v", err)
		}
	}

	result := map[string]interface{}{
		"changes": changes,
		"dryRun":  dryRun,
	}

	resultData, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("序列化结果失败: %v", err)
	}

	return mcp.NewToolResultText(string(resultData)), nil
}

func deleteDirs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["path"] == nil {
		return nil, errors.New("path 不能为空")
	}
	path := request.Params.Arguments["path"].(string)
	err := os.RemoveAll(path)
	if err != nil {
		if os.IsNotExist(err) {
			return mcp.NewToolResultText("目录已经不存在"), nil
		}
		return nil, err
	}
	return mcp.NewToolResultText("目录删除成功"), nil
}

func deleteFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["file"] == nil {
		return nil, errors.New("file 不能为空")
	}
	file := request.Params.Arguments["file"].(string)
	err := os.Remove(file)
	if err != nil {
		if os.IsNotExist(err) {
			return mcp.NewToolResultText("文件已不存在"), nil
		}
		return nil, err
	}
	return nil, err
}

func command(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if "" == request.Params.Arguments["command"] || nil == request.Params.Arguments["command"] {
		return nil, errors.New("command 不能为空")
	}
	workDir := ""
	if "" != request.Params.Arguments["workDir"] && nil != request.Params.Arguments["workDir"] {
		workDir = request.Params.Arguments["workDir"].(string)
	}

	if "" == request.Params.Arguments["wait"] {
		return nil, errors.New("wait 不能为空")
	}

	wait, ok := request.Params.Arguments["wait"].(bool)
	if !ok {
		return nil, errors.New("wait 类型为boolean")
	}

	var cmdTool string
	switch runtime.GOOS {
	case "windows":
		cmdTool = "C:\\Windows\\System32\\cmd.exe"
	case "linux":
		cmdTool = "/bin/bash"
	}
	xcmd := request.Params.Arguments["command"].(string)
	cmd := exec.Cmd{
		Path: cmdTool,
		Args: []string{"/c", xcmd},
		Dir:  workDir,
	}

	if wait {
		output, err := cmd.Output()
		if err != nil {
			return nil, err
		}

		s, err := simplifiedchinese.GBK.NewDecoder().String(string(output))
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(s), nil
	} else {
		err := cmd.Start()
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText("执行成功"), nil
	}
}
