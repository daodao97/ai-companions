package xtools

import (
	"companions/internal/pkg/xllm"
	"context"
	"time"
)

type TimeTool struct {
	Schema xllm.Tool
}

func NewTimeTool() ToolInterface {
	return &TimeTool{
		Schema: xllm.Tool{
			Name:        "time",
			Description: "获取当前时间",
		},
	}
}

func (t *TimeTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	return time.Now().Format("2006-01-02 15:04:05"), nil
}

func (t *TimeTool) GetSchema() xllm.Tool {
	return t.Schema
}
