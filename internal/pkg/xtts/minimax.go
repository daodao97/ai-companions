package xtts

import (
	"companions/internal/conf"
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/daodao97/xgo/xlog"
	"github.com/daodao97/xgo/xrequest"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
)

type Minimax struct {
	GroupID string
	APIKey  string
	Config  *conf.TTSConfig
}

type MinimaxTTSRequest struct {
	Model             string            `json:"model"`
	Text              string            `json:"text"`
	Stream            bool              `json:"stream"`
	VoiceSetting      VoiceSetting      `json:"voice_setting"`
	PronunciationDict PronunciationDict `json:"pronunciation_dict,omitempty"`
	AudioSetting      AudioSetting      `json:"audio_setting"`
	TimberWeights     []TimberWeights   `json:"timber_weights,omitempty"`
}

type TimberWeights struct {
	VoiceID string `json:"voice_id"`
	Weight  int    `json:"weight"`
}

type VoiceSetting struct {
	VoiceID string  `json:"voice_id"`
	Speed   float64 `json:"speed"`
	Vol     float64 `json:"vol"`
	Pitch   int     `json:"pitch"`
}

type PronunciationDict struct {
	Tone []string `json:"tone,omitempty"`
}

type AudioSetting struct {
	SampleRate int    `json:"sample_rate"`
	Bitrate    int    `json:"bitrate"`
	Format     string `json:"format"`
	Channel    int    `json:"channel"`
}

func NewMinimax(ttsConf *conf.TTSConfig) *Minimax {
	return &Minimax{
		GroupID: ttsConf.GroupId,
		APIKey:  ttsConf.ApiKey,
		Config:  ttsConf,
	}
}

func (m *Minimax) buildTTSRequest(req AudioReq) MinimaxTTSRequest {
	if req.Model == "" {
		req.Model = m.Config.Model
	}
	if req.Voice == "" {
		req.Voice = m.Config.Voice
	}
	if req.Speed == "" {
		req.Speed = m.Config.Speed
	}
	if req.Vol == "" {
		req.Vol = m.Config.Volume
	}
	if req.Pitch == "" {
		req.Pitch = m.Config.Pitch
	}
	if req.Format == "" {
		req.Format = m.Config.Format
	}

	var timberWeights []TimberWeights

	voiceSetting := VoiceSetting{
		VoiceID: req.Voice,
		Speed:   cast.ToFloat64(req.Speed),
		Vol:     cast.ToFloat64(req.Vol),
		Pitch:   cast.ToInt(req.Pitch),
	}

	if strings.Contains(req.Voice, "(") {
		voiceSetting.VoiceID = ""
		timberWeights = []TimberWeights{
			{
				VoiceID: req.Voice,
				Weight:  1,
			},
		}
	}

	return MinimaxTTSRequest{
		Model:        req.Model,
		Text:         req.Text,
		Stream:       true,
		VoiceSetting: voiceSetting,
		AudioSetting: AudioSetting{
			SampleRate: 32000,
			Bitrate:    128000,
			Format:     req.Format,
			Channel:    1,
		},
		TimberWeights: timberWeights,
	}
}

func (m *Minimax) TextToSpeech(req AudioReq) (AudioStream, error) {
	return m.TextToSpeechWithContext(context.Background(), req)
}

func (m *Minimax) TextToSpeechWithContext(ctx context.Context, req AudioReq) (AudioStream, error) {
	if req.Text == "" {
		return nil, errors.New("text is required")
	}

	if m.GroupID == "" || m.APIKey == "" {
		return nil, errors.New("GroupID and APIKey are required")
	}

	audioStream := make(AudioStream, 100)

	go func() {
		defer close(audioStream)

		url := fmt.Sprintf("https://api.minimaxi.com/v1/t2a_v2?GroupId=%s", m.GroupID)
		requestBody := m.buildTTSRequest(req)

		request, err := xrequest.New().
			SetDebug(false).
			SetHeader("Authorization", "Bearer "+m.APIKey).
			SetHeader("Content-Type", "application/json").
			SetHeader("Accept", "application/json, text/plain, */*").
			SetBody(requestBody).
			Post(url)

		if err != nil {
			xlog.ErrorC(ctx, "TTS请求失败: %v", err)
			return
		}

		if err := request.Error(); err != nil {
			xlog.ErrorC(ctx, "TTS请求错误: %v", err)
			return
		}

		stream, err := request.Stream()
		if err != nil {
			xlog.ErrorC(ctx, "获取流失败: %v", err)
			return
		}

		m.processStream(ctx, stream, audioStream)
		xlog.InfoC(ctx, "processStream 方法执行完成，goroutine 即将结束")
	}()

	return audioStream, nil
}

func (m *Minimax) processStream(ctx context.Context, stream chan string, audioStream AudioStream) {
	for {
		select {
		case <-ctx.Done():
			xlog.InfoC(ctx, "TTS流处理被取消")
			return
		case data, ok := <-stream:
			if !ok {
				xlog.InfoC(ctx, "TTS流处理完成")
				return
			}

			// 跳过空行和非 data: 开头的行
			if !strings.HasPrefix(data, "data:") {
				continue
			}

			// 移除 "data:" 前缀
			jsonData := strings.TrimSpace(strings.TrimPrefix(data, "data:"))

			if jsonData == "" || jsonData == "[DONE]" {
				continue
			}

			// 解析 JSON 响应
			result := gjson.Parse(jsonData)

			// 检查base_resp状态
			if baseResp := result.Get("base_resp"); baseResp.Exists() {
				statusCode := baseResp.Get("status_code").Int()
				if statusCode != 0 {
					statusMsg := baseResp.Get("status_msg").String()
					xlog.ErrorC(ctx, "API错误: %s (code: %d)", statusMsg, statusCode)
					continue
				}
			}

			if result.Get("data.status").Int() == 2 {
				if extraInfo := result.Get("extra_info"); extraInfo.Exists() {
					audioLength := extraInfo.Get("audio_length").Int()
					audioSize := extraInfo.Get("audio_size").Int()
					usageCharacters := extraInfo.Get("usage_characters").Int()
					xlog.InfoC(ctx, "TTS完成 - 音频长度: %d ms, 大小: %d bytes, 字符数: %d",
						audioLength, audioSize, usageCharacters)
				}
				xlog.InfoC(ctx, "TTS状态为2，processStream 即将返回")
				return
			}

			// 处理音频数据
			if audioData := result.Get("data.audio"); audioData.Exists() && audioData.String() != "" {
				audioHex := audioData.String()

				// 将十六进制字符串解码为字节
				audioBytes, err := hex.DecodeString(audioHex)
				if err != nil {
					xlog.ErrorC(ctx, "音频数据解码失败: %v", err)
					continue
				}

				// 将音频字节转换为base64编码，方便前端直接使用
				base64Data := base64.StdEncoding.EncodeToString(audioBytes)

				// 创建AudioChunk，使用base64编码的数据
				chunk := AudioChunk{
					Data:   base64Data, // 存储base64编码的字符串
					Format: "mp3",
				}

				select {
				case audioStream <- chunk:
					xlog.DebugC(ctx, "发送音频块: %d bytes (base64编码)", len(base64Data))
				case <-ctx.Done():
					return
				}

				// 如果是结束chunk，打印额外信息并结束

			}
		}
	}
}
