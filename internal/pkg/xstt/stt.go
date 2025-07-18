package xstt

import (
	"companions/internal/conf"
	"context"
)

type STT interface {
	SpeechToText(ctx context.Context, req SpeechToTextReq) (*SpeechToTextResp, error)
}

type SpeechToTextReq struct {
	Audio  string `json:"audio"`
	Format string `json:"format"`
}

type SpeechToTextResp struct {
	Text string `json:"text"`
}

func New(sttConf *conf.STTConfig) STT {
	switch sttConf.Provider {
	case "openai":
		return NewOpenAI(
			WithAPIKey(sttConf.ApiKey),
			WithAPIUrl(sttConf.ApiUrl),
			WithModel(sttConf.Model),
		)
	}
	return nil
}
