package xagent

import (
	"companions/internal/pkg/xllm"
	"context"
	"encoding/json"
)

type Input map[string]any

func (i Input) Scan(dest any) error {
	return json.Unmarshal([]byte(i.ToJSON()), dest)
}

func (i Input) ToJSON() string {
	json, err := json.Marshal(i)
	if err != nil {
		return ""
	}
	return string(json)
}

func NewInput(data map[string]any) Input {
	return Input(data)
}

func NewInputFromJSON(data string) (Input, error) {
	var input Input
	err := json.Unmarshal([]byte(data), &input)
	return input, err
}

// Agent 定义了所有 Agent 的公共接口
type Agent interface {
	// GetName 获取 Agent 名称
	GetName() string

	// GetInstructions 获取 Agent 指令
	GetInstructions() string

	// GetLLM 获取使用的 LLM
	GetLLM() xllm.LLM

	// Execute 执行任务，返回结构化消息流
	Execute(ctx context.Context, input Input) (chan Message, error)

	// Schema
	GetSchema() xllm.Tool
}

// BaseAgent 基础 Agent 结构，包含公共字段
type BaseAgent struct {
	name         string
	instructions string
	llm          xllm.LLM
	schema       []xllm.Parameter
}

// NewBaseAgent 创建基础 Agent
func NewBaseAgent(name, instructions string, llm xllm.LLM, schema []xllm.Parameter) *BaseAgent {
	return &BaseAgent{
		name:         name,
		instructions: instructions,
		llm:          llm,
		schema:       schema,
	}
}

// GetName 实现 Agent 接口
func (b *BaseAgent) GetName() string {
	return b.name
}

// GetInstructions 实现 Agent 接口
func (b *BaseAgent) GetInstructions() string {
	return b.instructions
}

// GetLLM 实现 Agent 接口
func (b *BaseAgent) GetLLM() xllm.LLM {
	return b.llm
}

// GetSchema 返回工具的 schema, 用于 function call
func (b *BaseAgent) GetSchema() xllm.Tool {
	return xllm.Tool{
		Name:        b.name,
		Description: b.instructions,
		Parameters:  b.schema,
	}
}
