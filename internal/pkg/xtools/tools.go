package xtools

import (
	"context"
	"fmt"

	"companions/internal/pkg/xllm"
)

type ToolInterface interface {
	GetSchema() xllm.Tool
	Execute(ctx context.Context, args map[string]any) (string, error)
}

type Tool struct {
	Schema  xllm.Tool
	Execute func(ctx context.Context, args map[string]any) (string, error)
}

type Tools struct {
	tools []ToolInterface
}

func NewTools(tools ...ToolInterface) *Tools {
	return &Tools{
		tools: tools,
	}
}

func (x *Tools) GetTools() []xllm.Tool {
	tools := make([]xllm.Tool, len(x.tools))
	for i, tool := range x.tools {
		tools[i] = tool.GetSchema()
	}
	return tools
}

func (x *Tools) AddTool(tool ...ToolInterface) {
	x.tools = append(x.tools, tool...)
}

func (x *Tools) CallTool(ctx context.Context, name string, args map[string]any) (string, error) {
	for _, tool := range x.tools {
		if tool.GetSchema().Name == name {
			return tool.Execute(ctx, args)
		}
	}
	return "", fmt.Errorf("tool not found")
}
