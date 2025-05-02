package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gocv.io/x/gocv"
)

// VideoEncoder 视频编码器结构体
type VideoEncoder struct {
	inputImages []string
	outputVideo string
	frameRate   float64
}

// NewVideoEncoder 创建新的视频编码器
func NewVideoEncoder(inputImages []string, outputVideo string, frameRate float64) *VideoEncoder {
	return &VideoEncoder{
		inputImages: inputImages,
		outputVideo: outputVideo,
		frameRate:   frameRate,
	}
}

// ValidateInput 验证输入参数
func (ve *VideoEncoder) ValidateInput() error {
	if len(ve.inputImages) == 0 {
		return fmt.Errorf("输入图片列表不能为空")
	}

	for _, imgPath := range ve.inputImages {
		if _, err := os.Stat(imgPath); os.IsNotExist(err) {
			return fmt.Errorf("图片文件不存在: %s", imgPath)
		}
	}

	outputDir := filepath.Dir(ve.outputVideo)
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		return fmt.Errorf("输出目录不存在: %s", outputDir)
	}

	if ve.frameRate <= 0 {
		return fmt.Errorf("帧率必须大于0")
	}

	return nil
}

// Encode 编码视频
func (ve *VideoEncoder) Encode() error {
	log.Printf("开始视频编码处理，共%d张图片", len(ve.inputImages))

	// 读取第一张图片获取尺寸
	firstImg := gocv.IMRead(ve.inputImages[0], gocv.IMReadColor)
	if firstImg.Empty() {
		log.Printf("错误: 无法读取第一张图片: %s", ve.inputImages[0])
		return fmt.Errorf("无法读取第一张图片: %s", ve.inputImages[0])
	}
	defer firstImg.Close()
	log.Printf("成功读取第一张图片，尺寸: %dx%d", firstImg.Cols(), firstImg.Rows())

	// 创建视频写入器
	log.Printf("正在创建视频写入器，输出路径: %s，帧率: %.2f", ve.outputVideo, ve.frameRate)
	writer, err := gocv.VideoWriterFile(
		ve.outputVideo,
		"mp4v",
		ve.frameRate,
		firstImg.Cols(),
		firstImg.Rows(),
		true)
	if err != nil {
		log.Printf("错误: 无法创建视频写入器: %v", err)
		return fmt.Errorf("无法创建视频写入器: %v", err)
	}
	defer writer.Close()
	log.Printf("视频写入器创建成功")

	// 处理所有图片
	total := len(ve.inputImages)
	for i, imgPath := range ve.inputImages {
		log.Printf("正在处理第%d/%d张图片: %s", i+1, total, imgPath)
		img := gocv.IMRead(imgPath, gocv.IMReadColor)
		if img.Empty() {
			log.Printf("警告: 无法读取图片 %s，跳过", imgPath)
			continue
		}

		if img.Cols() != firstImg.Cols() || img.Rows() != firstImg.Rows() {
			log.Printf("警告: 图片 %s 尺寸不匹配(实际: %dx%d，期望: %dx%d)，跳过",
				imgPath, img.Cols(), img.Rows(), firstImg.Cols(), firstImg.Rows())
			img.Close()
			continue
		}

		writer.Write(img)
		img.Close()
		log.Printf("成功处理第%d/%d张图片", i+1, total)
	}

	log.Printf("视频编码完成，共处理%d张图片", total)
	return nil
}

// VideoEncoderResult 视频编码结果结构体
type VideoEncoderResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
