package xtts

import "companions/internal/conf"

type AudioChunk struct {
	Data   string `json:"data"`
	Format string `json:"format"`
}

type AudioReq struct {
	Text   string `json:"text"`   // 文本
	Model  string `json:"model"`  // 模型
	Voice  string `json:"voice"`  // 语音
	Speed  string `json:"speed"`  // 速度
	Vol    string `json:"vol"`    // 音量
	Pitch  string `json:"pitch"`  // 音调
	Format string `json:"format"` // 格式
}

type AudioStream chan AudioChunk

type TTS interface {
	TextToSpeech(req AudioReq) (AudioStream, error)
}

func New(ttsConf *conf.TTSConfig) TTS {
	switch ttsConf.Provider {
	case "minimax":
		return NewMinimax(ttsConf)
	}
	return nil
}
