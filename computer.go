package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"os/exec"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"github.com/go-vgo/robotgo"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
)

func addComputerTools(mcpServer *server.MCPServer) {
	mcpServer.AddTool(mcp.NewTool("window_screen", mcp.WithDescription("获取屏幕当前展示内容生成图片")), windowScreen)

	mcpServer.AddTool(mcp.NewTool("mouse_move",
		mcp.WithDescription("鼠标移动"),
		mcp.WithNumber("x", mcp.Description("鼠标将要移动的X坐标"), mcp.Required()),
		mcp.WithNumber("y", mcp.Description("鼠标将要移动的Y坐标"), mcp.Required())),
		mouseMove)
	mcpServer.AddTool(mcp.NewTool("mouse_click",
		mcp.WithDescription("鼠标点击"),
		mcp.WithString("time", mcp.Description("点击次数"), mcp.Required()),
		mcp.WithString("key", mcp.Description("鼠标键"), mcp.Required())),
		mouseClick)

	mcpServer.AddTool(mcp.NewTool("current_time", mcp.WithDescription("当前时间")), currentTime)

	// 窗口管理工具
	mcpServer.AddTool(mcp.NewTool("window_list",
		mcp.WithDescription("获取窗口列表")),
		windowList)
	mcpServer.AddTool(mcp.NewTool("window_activate",
		mcp.WithDescription("激活指定窗口"),
		mcp.WithString("title", mcp.Description("窗口标题"), mcp.Required())),
		windowActivate)

	// 键盘输入工具
	mcpServer.AddTool(mcp.NewTool("keyboard_type",
		mcp.WithDescription("键盘输入文本"),
		mcp.WithString("text", mcp.Description("要输入的文本"), mcp.Required())),
		keyboardType)
	mcpServer.AddTool(mcp.NewTool("keyboard_press",
		mcp.WithDescription("按下键盘快捷键"),
		mcp.WithString("keys", mcp.Description("快捷键组合(用+分隔)"), mcp.Required())),
		keyboardPress)

	// 系统监控工具
	mcpServer.AddTool(mcp.NewTool("system_cpu_usage",
		mcp.WithDescription("获取CPU使用率")),
		getCpuUsage)
	mcpServer.AddTool(mcp.NewTool("system_memory_usage",
		mcp.WithDescription("获取内存使用情况")),
		getMemoryUsage)

	// 进程管理工具
	mcpServer.AddTool(mcp.NewTool("process_list",
		mcp.WithDescription("获取进程列表")),
		getProcessList)
	mcpServer.AddTool(mcp.NewTool("process_kill",
		mcp.WithDescription("终止进程"),
		mcp.WithNumber("pid", mcp.Description("进程ID"), mcp.Required())),
		killProcess)

	// 系统信息监控工具
	mcpServer.AddTool(mcp.NewTool("network_status",
		mcp.WithDescription("获取网络状态信息")),
		getNetworkStatus)
	mcpServer.AddTool(mcp.NewTool("disk_io",
		mcp.WithDescription("获取磁盘IO信息")),
		getDiskIO)

	// 剪贴板工具
	mcpServer.AddTool(mcp.NewTool("clipboard_get",
		mcp.WithDescription("获取剪贴板内容")),
		getClipboard)
	mcpServer.AddTool(mcp.NewTool("clipboard_set",
		mcp.WithDescription("设置剪贴板内容"),
		mcp.WithString("text", mcp.Description("要设置的文本内容"), mcp.Required())),
		setClipboard)

	// 音量控制工具
	mcpServer.AddTool(mcp.NewTool("volume_get",
		mcp.WithDescription("获取当前系统音量")),
		getVolume)
	mcpServer.AddTool(mcp.NewTool("volume_set",
		mcp.WithDescription("设置系统音量"),
		mcp.WithNumber("level", mcp.Description("音量级别(0-100)"), mcp.Required())),
		setVolume)

	// 电源管理工具
	mcpServer.AddTool(mcp.NewTool("computer_shutdown",
		mcp.WithDescription("关闭计算机")),
		shutdownComputer)
	mcpServer.AddTool(mcp.NewTool("computer_restart",
		mcp.WithDescription("重启计算机")),
		restartComputer)
	mcpServer.AddTool(mcp.NewTool("computer_sleep",
		mcp.WithDescription("使计算机进入睡眠模式")),
		sleepComputer)
}

func currentTime(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(time.Now().Format(time.DateTime)), nil
}

func windowScreen(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	w, h := robotgo.GetScreenSize()
	log.Printf("width: %d, height: %d\n", w, h)
	fileName := uuid.NewString() + ".png"
	err := robotgo.SaveCapture(fileName, 0, 0, w, h)
	if err != nil {
		return nil, err
	}

	abs, err := filepath.Abs(fileName)
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText("截图成功，浏览器打开file://" + abs), nil
}

func mouseMove(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Println(request.Params.Arguments)
	if request.Params.Arguments["x"] == nil {
		return nil, errors.New("x 参数不能为空")
	}
	if request.Params.Arguments["y"] == nil {
		return nil, errors.New("y 参数不能为空")
	}

	x, err := strconv.Atoi(fmt.Sprintf("%.0f", request.Params.Arguments["x"].(float64)))
	if err != nil {
		return nil, err
	}
	y, err := strconv.Atoi(fmt.Sprintf("%.0f", request.Params.Arguments["y"].(float64)))
	if err != nil {
		return nil, err
	}
	robotgo.MouseSleep = 100
	w, h := robotgo.GetScreenSize()
	log.Printf("width: %d, height: %d\n", w, h)
	robotgo.Move(x, h-y)

	return mcp.NewToolResultText("移动成功"), nil
}

func mouseClick(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Println(request.Params.Arguments)

	if request.Params.Arguments["time"] == nil {
		return nil, errors.New("time 参数不能为空")
	}
	if request.Params.Arguments["key"] == nil {
		return nil, errors.New("key 参数不能为空")
	}
	time := request.Params.Arguments["time"].(string)
	key := request.Params.Arguments["key"].(string)
	doubleClick := true
	if time == "1" {
		doubleClick = false
	}
	robotgo.Click(key, doubleClick)
	log.Printf("鼠标点击, 键: %s, 次数: %s\n", key, time)

	return mcp.NewToolResultText("点击成功"), nil
}

// windowList 获取窗口列表
func windowList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	windows, err := robotgo.FindIds("")
	if err != nil {
		return nil, err
	}
	output := "窗口列表:\n"
	for i, pid := range windows {
		title := robotgo.GetTitle(pid)
		output += fmt.Sprintf("%d. PID: %d, 标题: %s\n", i+1, pid, title)
	}
	return mcp.NewToolResultText(output), nil
}

// windowActivate 激活指定窗口
func windowActivate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["title"] == nil {
		return nil, errors.New("title 参数不能为空")
	}
	title := request.Params.Arguments["title"].(string)
	pid := robotgo.FindWindow(title)
	if pid == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("未找到标题为 '%s' 的窗口", title)), nil
	}
	robotgo.SetActiveWindow(pid)
	return mcp.NewToolResultText(fmt.Sprintf("已激活窗口: %s", title)), nil
}

// keyboardType 键盘输入文本
func keyboardType(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["text"] == nil {
		return nil, errors.New("text 参数不能为空")
	}
	text := request.Params.Arguments["text"].(string)
	robotgo.TypeStr(text)
	return mcp.NewToolResultText(fmt.Sprintf("已输入文本: %s", text)), nil
}

// keyboardPress 按下键盘快捷键
func keyboardPress(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["keys"] == nil {
		return nil, errors.New("keys 参数不能为空")
	}
	keys := request.Params.Arguments["keys"].(string)
	keyList := strings.Split(keys, "+")
	for _, key := range keyList {
		robotgo.KeyTap(key)
	}
	return mcp.NewToolResultText(fmt.Sprintf("已按下快捷键: %s", keys)), nil
}

// getCpuUsage 获取CPU使用率
func getCpuUsage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	percent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return nil, fmt.Errorf("获取CPU使用率失败: %v", err)
	}
	if len(percent) == 0 {
		return nil, errors.New("无法获取CPU使用率")
	}
	return mcp.NewToolResultText(fmt.Sprintf("CPU使用率: %.2f%%", percent[0])), nil
}

// getMemoryUsage 获取内存使用情况
func getMemoryUsage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mem, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("获取内存信息失败: %v", err)
	}
	output := fmt.Sprintf("内存使用情况:\n"+
		"总内存: %.2f MB\n"+
		"已使用: %.2f MB\n"+
		"空闲: %.2f MB\n"+
		"缓存: %.2f MB\n"+
		"使用率: %.2f%%",
		float64(mem.Total)/1024/1024,
		float64(mem.Used)/1024/1024,
		float64(mem.Free)/1024/1024,
		float64(mem.Cached)/1024/1024,
		mem.UsedPercent)
	return mcp.NewToolResultText(output), nil
}

// getProcessList 获取进程列表
func getProcessList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("获取进程列表失败: %v", err)
	}

	output := "进程列表:\n"
	for i, p := range processes {
		name, _ := p.Name()
		output += fmt.Sprintf("%d. PID: %d, 名称: %s\n", i+1, p.Pid, name)
	}
	return mcp.NewToolResultText(output), nil
}

// killProcess 终止进程
func killProcess(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["pid"] == nil {
		return nil, errors.New("pid 参数不能为空")
	}
	pid := int32(request.Params.Arguments["pid"].(float64))
	p, err := os.FindProcess(int(pid))
	if err != nil {
		return nil, fmt.Errorf("查找进程失败: %v", err)
	}
	err = p.Kill()
	if err != nil {
		return nil, fmt.Errorf("终止进程失败: %v", err)
	}
	return mcp.NewToolResultText(fmt.Sprintf("已终止进程: %d", pid)), nil
}

// getNetworkStatus 获取网络状态信息
func getNetworkStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	netInfo, err := net.IOCounters(true)
	if err != nil {
		return nil, fmt.Errorf("获取网络信息失败: %v", err)
	}

	output := "网络状态:\n"
	for _, info := range netInfo {
		output += fmt.Sprintf("接口: %s\n"+
			"接收字节: %d\n"+
			"发送字节: %d\n"+
			"接收包数: %d\n"+
			"发送包数: %d\n",
			info.Name, info.BytesRecv, info.BytesSent, info.PacketsRecv, info.PacketsSent)
	}
	return mcp.NewToolResultText(output), nil
}

// getDiskIO 获取磁盘IO信息
func getDiskIO(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	diskInfo, err := disk.IOCounters()
	if err != nil {
		return nil, fmt.Errorf("获取磁盘IO信息失败: %v", err)
	}

	output := "磁盘IO信息:\n"
	for name, info := range diskInfo {
		output += fmt.Sprintf("磁盘: %s\n"+
			"读取次数: %d\n"+
			"写入次数: %d\n"+
			"读取字节: %d\n"+
			"写入字节: %d\n"+
			"读取时间(ms): %d\n"+
			"写入时间(ms): %d\n",
			name, info.ReadCount, info.WriteCount, info.ReadBytes, info.WriteBytes, info.ReadTime, info.WriteTime)
	}
	return mcp.NewToolResultText(output), nil
}

// getClipboard 获取剪贴板内容
func getClipboard(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	text, err := robotgo.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("获取剪贴板内容失败: %v", err)
	}
	return mcp.NewToolResultText(fmt.Sprintf("剪贴板内容: %s", text)), nil
}

// setClipboard 设置剪贴板内容
func setClipboard(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["text"] == nil {
		return nil, errors.New("text 参数不能为空")
	}
	text := request.Params.Arguments["text"].(string)
	err := robotgo.WriteAll(text)
	if err != nil {
		return nil, fmt.Errorf("设置剪贴板内容失败: %v", err)
	}
	return mcp.NewToolResultText("剪贴板内容已设置"), nil
}

// getVolume 获取当前系统音量
func getVolume(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ole.CoInitialize(0)
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("WMPlayer.OCX")
	if err != nil {
		return nil, fmt.Errorf("创建WMPlayer对象失败: %v", err)
	}
	defer unknown.Release()

	wmp, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, fmt.Errorf("获取WMPlayer接口失败: %v", err)
	}
	defer wmp.Release()

	player := oleutil.MustCallMethod(wmp, "settings").ToIDispatch()
	defer player.Release()

	vol := oleutil.MustGetProperty(player, "volume").Val
	return mcp.NewToolResultText(fmt.Sprintf("当前音量: %d%%", int(vol))), nil
}

// setVolume 设置系统音量
func setVolume(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.Params.Arguments["level"] == nil {
		return nil, errors.New("level 参数不能为空")
	}
	level := int(request.Params.Arguments["level"].(float64))
	if level < 0 || level > 100 {
		return nil, errors.New("音量级别必须在0-100之间")
	}

	ole.CoInitialize(0)
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("WMPlayer.OCX")
	if err != nil {
		return nil, fmt.Errorf("创建WMPlayer对象失败: %v", err)
	}
	defer unknown.Release()

	wmp, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, fmt.Errorf("获取WMPlayer接口失败: %v", err)
	}
	defer wmp.Release()

	player := oleutil.MustCallMethod(wmp, "settings").ToIDispatch()
	defer player.Release()

	_, err = oleutil.PutProperty(player, "volume", level)
	if err != nil {
		return nil, fmt.Errorf("设置音量失败: %v", err)
	}
	return mcp.NewToolResultText(fmt.Sprintf("音量已设置为: %d%%", level)), nil
}

// shutdownComputer 关闭计算机
func shutdownComputer(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cmd := exec.Command("shutdown", "/s", "/t", "60")
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("关闭计算机失败: %v", err)
	}
	return mcp.NewToolResultText("计算机将在60秒后关闭"), nil
}

// restartComputer 重启计算机
func restartComputer(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cmd := exec.Command("shutdown", "/r", "/t", "60")
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("重启计算机失败: %v", err)
	}
	return mcp.NewToolResultText("计算机将在60秒后重启"), nil
}

// sleepComputer 使计算机进入睡眠模式
func sleepComputer(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cmd := exec.Command("rundll32.exe", "powrprof.dll,SetSuspendState", "0,1,0")
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("进入睡眠模式失败: %v", err)
	}
	return mcp.NewToolResultText("计算机将进入睡眠模式"), nil
}
