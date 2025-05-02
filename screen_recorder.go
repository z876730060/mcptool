package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var (
	// 定义全局变量
	recordingTasks = make(map[string]*ScreenRecorder)
)

// ScreenRecorder 屏幕录制器结构体
type ScreenRecorder struct {
	taskID      string
	frameRate   float64
	resolution  string
	outputDir   string
	isRecording bool
	stopChan    chan struct{}
}

// NewScreenRecorder 创建新的屏幕录制器
func NewScreenRecorder(taskID string, frameRate float64, resolution string, outputDir string) *ScreenRecorder {
	return &ScreenRecorder{
		taskID:      taskID,
		frameRate:   frameRate,
		resolution:  resolution,
		outputDir:   outputDir,
		isRecording: false,
		stopChan:    make(chan struct{}),
	}
}

// StartRecording 开始屏幕录制
func (sr *ScreenRecorder) StartRecording(w, h int) ([]string, error) {
	if sr.isRecording {
		return nil, fmt.Errorf("录制任务已在运行中")
	}

	sr.isRecording = true
	var imagePaths []string

	// 创建输出目录
	if err := os.MkdirAll(sr.outputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %v", err)
	}

	// 开始录制
	go func() {
		ticker := time.NewTicker(time.Duration(1000/sr.frameRate) * time.Millisecond)
		defer ticker.Stop()

		frameCount := 0
		for {
			select {
			case <-ticker.C:
				imgPath := filepath.Join(sr.outputDir, fmt.Sprintf("frame_%04d.png", frameCount))
				err := robotgo.SaveCapture(imgPath, 0, 0, w, h)
				if err != nil {
					log.Println(fmt.Errorf("保存截图失败: %v", err))
				}
				frameCount++
				imagePaths = append(imagePaths, imgPath)
			case <-sr.stopChan:
				return
			}
		}
	}()

	return imagePaths, nil
}

// StopRecording 停止屏幕录制
func (sr *ScreenRecorder) StopRecording() {
	if sr.isRecording {
		close(sr.stopChan)
		sr.isRecording = false
	}
}

// ScreenRecorderResult 屏幕录制结果结构体
type ScreenRecorderResult struct {
	TaskID     string   `json:"task_id"`
	ImagePaths []string `json:"image_paths"`
	Message    string   `json:"message"`
}

// startRecording 启动屏幕录制
func startRecording(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	w, h := robotgo.GetScreenSize()
	log.Printf("width: %d, height: %d\n", w, h)

	frameRate := 30.0
	if request.Params.Arguments["frame_rate"] != nil {
		frameRate = request.Params.Arguments["frame_rate"].(float64)
	}

	// 生成任务ID
	taskID := generateTaskID()
	outputDir := filepath.Join("recordings", taskID)

	recorder := NewScreenRecorder(taskID, frameRate, fmt.Sprintf("%d*%d", w, h), outputDir)
	recordingTasks[taskID] = recorder
	_, err := recorder.StartRecording(w, h)
	if err != nil {
		result := ScreenRecorderResult{
			TaskID:  taskID,
			Message: fmt.Sprintf("启动录制失败: %v", err),
		}
		return mcp.NewToolResultText(EncodeSerializedData(result)), nil
	}

	result := ScreenRecorderResult{
		TaskID:  taskID,
		Message: "录制已启动",
	}
	return mcp.NewToolResultText(EncodeSerializedData(result)), nil
}

// stopRecording 停止屏幕录制
func stopRecording(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	taskID := request.Params.Arguments["task_id"].(string)
	outputDir := filepath.Join("recordings", taskID)
	log.Printf("开始停止录制任务: %s, 输出目录: %s", taskID, outputDir)

	recorder, exists := recordingTasks[taskID]
	if !exists {
		log.Printf("录制任务不存在: taskID=%s, outputDir=%s", taskID, outputDir)
		result := ScreenRecorderResult{
			TaskID:  taskID,
			Message: fmt.Sprintf("录制任务 %s 不存在", taskID),
		}
		return mcp.NewToolResultText(EncodeSerializedData(result)), nil
	}

	log.Printf("正在停止录制任务: %s", taskID)
	recorder.StopRecording()
	delete(recordingTasks, taskID)
	log.Printf("已停止录制任务: %s", taskID)

	// 获取所有图片文件
	log.Printf("正在获取录制图片文件: %s", outputDir)
	files, err := filepath.Glob(filepath.Join(outputDir, "*.png"))
	if err != nil {
		log.Printf("获取图片文件失败: %v, 目录: %s", err, outputDir)
		result := ScreenRecorderResult{
			TaskID:  taskID,
			Message: fmt.Sprintf("获取图片文件失败: %v", err),
		}
		return mcp.NewToolResultText(EncodeSerializedData(result)), nil
	}
	log.Printf("成功获取 %d 个图片文件", len(files))

	encoder := NewVideoEncoder(files, fmt.Sprintf("%s.mp4", outputDir), recorder.frameRate)
	result := &VideoEncoderResult{}

	// 验证输入
	log.Printf("正在验证视频输入文件")
	if err := encoder.ValidateInput(); err != nil {
		log.Printf("输入验证失败: %v", err)
		result.Success = false
		result.Message = fmt.Sprintf("输入验证失败: %v", err)
		return mcp.NewToolResultText(EncodeSerializedData(result)), nil
	}

	// 编码视频
	log.Printf("开始视频编码, 帧率: %.2f", recorder.frameRate)
	if err := encoder.Encode(); err != nil {
		log.Printf("视频编码失败: %v", err)
		result.Success = false
		result.Message = fmt.Sprintf("视频编码失败: %v", err)
		return mcp.NewToolResultText(EncodeSerializedData(result)), nil
	}

	log.Printf("视频编码成功: %s", encoder.outputVideo)
	result.Success = true
	result.Message = fmt.Sprintf("视频编码成功: %s", encoder.outputVideo)

	return mcp.NewToolResultText(EncodeSerializedData(result)), nil
}

// generateTaskID 生成唯一任务ID
func generateTaskID() string {
	return fmt.Sprintf("rec_%d", time.Now().UnixNano())
}

// 添加屏幕录制工具
func addScreenRecorderTool(mcpServer *server.MCPServer) {
	mcpServer.AddTool(mcp.NewTool("start_recording",
		mcp.WithDescription("启动屏幕录制功能"),
		mcp.WithNumber("frame_rate", mcp.Required(), mcp.DefaultNumber(30.0), mcp.Description("录制帧率"))),
		startRecording)

	mcpServer.AddTool(mcp.NewTool("stop_recording",
		mcp.WithDescription("停止屏幕录制功能"),
		mcp.WithString("task_id", mcp.Required(), mcp.Description("录制任务ID"))),
		stopRecording)
}
