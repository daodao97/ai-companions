package xstt

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/daodao97/xgo/xlog"
	"github.com/daodao97/xgo/xrequest"
)

type OpenAI struct {
	APIKey  string
	APIUrl  string
	Model   string
	Timeout int    // 超时时间（秒）
	SaveDir string // 保存音频文件的目录
}

type OpenAIOption func(*OpenAI)

func WithAPIKey(apiKey string) OpenAIOption {
	return func(o *OpenAI) {
		o.APIKey = apiKey
	}
}

func WithAPIUrl(apiUrl string) OpenAIOption {
	return func(o *OpenAI) {
		o.APIUrl = apiUrl
	}
}

func WithModel(model string) OpenAIOption {
	return func(o *OpenAI) {
		o.Model = model
	}
}

func WithTimeout(timeout int) OpenAIOption {
	return func(o *OpenAI) {
		o.Timeout = timeout
	}
}

func WithSaveDir(saveDir string) OpenAIOption {
	return func(o *OpenAI) {
		o.SaveDir = saveDir
	}
}

func NewOpenAI(opts ...OpenAIOption) STT {
	openai := &OpenAI{
		APIUrl:  "https://api.gptsapi.net/v1",
		Model:   "whisper-1",
		Timeout: 30,
	}
	for _, opt := range opts {
		opt(openai)
	}
	return openai
}

func (o *OpenAI) SpeechToText(ctx context.Context, req SpeechToTextReq) (*SpeechToTextResp, error) {
	if o.APIKey == "" {
		return nil, errors.New("API key is required")
	}

	if req.Audio == "" {
		return nil, errors.New("audio data is required")
	}

	// 解码base64音频数据
	audioData, err := base64.StdEncoding.DecodeString(req.Audio)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio data: %v", err)
	}

	// 如果指定了保存目录，则保存音频文件
	var savedFilePath string
	if o.SaveDir != "" {
		// 确保保存目录存在
		if err := os.MkdirAll(o.SaveDir, 0755); err != nil {
			xlog.ErrorC(ctx, "创建保存目录失败: %v", err)
			return nil, fmt.Errorf("failed to create save directory: %v", err)
		}

		// 生成文件名（使用时间戳避免冲突）
		timestamp := time.Now().Format("20060102_150405_000")
		extension := req.Format
		if extension == "" {
			extension = "mp3" // 默认格式
		}
		fileName := fmt.Sprintf("audio_%s.%s", timestamp, extension)
		savedFilePath = filepath.Join(o.SaveDir, fileName)

		// 保存文件
		if err := os.WriteFile(savedFilePath, audioData, 0644); err != nil {
			xlog.ErrorC(ctx, "保存音频文件失败: %v", err)
			return nil, fmt.Errorf("failed to save audio file: %v", err)
		}

		xlog.InfoC(ctx, "音频文件已保存到: %s", savedFilePath)
	}

	// 发送请求
	url := o.APIUrl + "/audio/transcriptions"

	logMsg := fmt.Sprintf("发送STT请求到OpenAI: %s, 音频大小: %d bytes", url, len(audioData))
	if savedFilePath != "" {
		logMsg += fmt.Sprintf(", 已保存到: %s", savedFilePath)
	}
	xlog.InfoC(ctx, logMsg)

	request, err := xrequest.New().
		SetDebug(false).
		SetHeader("Authorization", "Bearer "+o.APIKey).
		SetFormData(map[string]string{
			"model":           o.Model,
			"response_format": "json",
		}).
		AddFile("file", "audio."+req.Format, bytes.NewReader(audioData)).
		Post(url)

	if err != nil {
		xlog.ErrorC(ctx, "STT请求失败: %v", err)
		return nil, fmt.Errorf("request failed: %v", err)
	}

	if err := request.Error(); err != nil {
		xlog.ErrorC(ctx, "STT请求错误: %v", err)
		return nil, fmt.Errorf("request error: %v", err)
	}

	// 解析响应
	response := request.Json()

	// 检查错误
	if errMsg := response.Get("error.message"); errMsg.Exists() {
		errorText := errMsg.String()
		xlog.ErrorC(ctx, "OpenAI API错误: %s", errorText)
		return nil, fmt.Errorf("OpenAI API error: %s", errorText)
	}

	// 获取转录文本
	text := response.Get("text").String()
	if text == "" {
		xlog.WarnC(ctx, "OpenAI返回空文本")
		return nil, errors.New("no text returned from OpenAI")
	}

	xlog.InfoC(ctx, "STT转换成功，文本长度: %d 字符", len(text))
	xlog.DebugC(ctx, "STT转换结果: %s", text)

	return &SpeechToTextResp{
		Text: text,
	}, nil
}

// SpeechToTextWithReader 使用io.Reader进行语音转文字
func (o *OpenAI) SpeechToTextWithReader(ctx context.Context, reader io.Reader, filename string) (*SpeechToTextResp, error) {
	if o.APIKey == "" {
		return nil, errors.New("API key is required")
	}

	// 读取音频数据
	audioData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio data: %v", err)
	}

	// 转换为base64并调用标准方法
	base64Data := base64.StdEncoding.EncodeToString(audioData)
	return o.SpeechToText(ctx, SpeechToTextReq{Audio: base64Data})
}

// SpeechToTextWithOptions 支持更多选项的语音转文字
func (o *OpenAI) SpeechToTextWithOptions(ctx context.Context, req SpeechToTextReq, options map[string]string) (*SpeechToTextResp, error) {
	if o.APIKey == "" {
		return nil, errors.New("API key is required")
	}

	if req.Audio == "" {
		return nil, errors.New("audio data is required")
	}

	// 解码base64音频数据
	audioData, err := base64.StdEncoding.DecodeString(req.Audio)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio data: %v", err)
	}

	// 创建multipart form数据
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 添加音频文件
	fileWriter, err := writer.CreateFormFile("file", "audio.mp3")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %v", err)
	}

	_, err = fileWriter.Write(audioData)
	if err != nil {
		return nil, fmt.Errorf("failed to write audio data: %v", err)
	}

	// 添加模型参数
	model := o.Model
	if modelOpt, exists := options["model"]; exists {
		model = modelOpt
	}
	err = writer.WriteField("model", model)
	if err != nil {
		return nil, fmt.Errorf("failed to write model field: %v", err)
	}

	// 添加其他可选参数
	supportedOptions := []string{
		"language",        // 输入音频的语言
		"prompt",          // 指导模型风格的可选文本
		"response_format", // 响应格式
		"temperature",     // 采样温度
	}

	for _, optionKey := range supportedOptions {
		if value, exists := options[optionKey]; exists {
			err = writer.WriteField(optionKey, value)
			if err != nil {
				return nil, fmt.Errorf("failed to write %s field: %v", optionKey, err)
			}
		}
	}

	// 如果没有指定响应格式，默认使用json
	if _, exists := options["response_format"]; !exists {
		err = writer.WriteField("response_format", "json")
		if err != nil {
			return nil, fmt.Errorf("failed to write response_format field: %v", err)
		}
	}

	// 关闭writer
	contentType := writer.FormDataContentType()
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %v", err)
	}

	// 发送请求
	url := o.APIUrl + "/audio/transcriptions"

	xlog.InfoC(ctx, "发送STT请求到OpenAI: %s, 音频大小: %d bytes, 选项: %+v", url, len(audioData), options)

	request, err := xrequest.New().
		SetDebug(false).
		SetHeader("Authorization", "Bearer "+o.APIKey).
		SetHeader("Content-Type", contentType).
		SetBody(buf.Bytes()).
		Post(url)

	if err != nil {
		xlog.ErrorC(ctx, "STT请求失败: %v", err)
		return nil, fmt.Errorf("request failed: %v", err)
	}

	if err := request.Error(); err != nil {
		xlog.ErrorC(ctx, "STT请求错误: %v", err)
		return nil, fmt.Errorf("request error: %v", err)
	}

	// 解析响应
	response := request.Json()

	// 检查错误
	if errMsg := response.Get("error.message"); errMsg.Exists() {
		errorText := errMsg.String()
		xlog.ErrorC(ctx, "OpenAI API错误: %s", errorText)
		return nil, fmt.Errorf("OpenAI API error: %s", errorText)
	}

	// 获取转录文本
	text := response.Get("text").String()
	if text == "" {
		xlog.WarnC(ctx, "OpenAI返回空文本")
		return nil, errors.New("no text returned from OpenAI")
	}

	xlog.InfoC(ctx, "STT转换成功，文本长度: %d 字符", len(text))
	xlog.DebugC(ctx, "STT转换结果: %s", text)

	return &SpeechToTextResp{
		Text: text,
	}, nil
}
