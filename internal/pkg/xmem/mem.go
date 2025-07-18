package xmem

import (
	"companions/internal/pkg/xllm"
)

type Conversation struct {
	Id        string `json:"id"`
	Title     string `json:"title"`
	Model     string `json:"model,omitempty"`
	Messages  []any  `json:"messages"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Memory interface {
	// 获取历史消息
	Get(convId string) (*Conversation, error)
	// 获取记忆
	GetMemory(convId string) ([]xllm.Message, error)
	// 设置历史消息
	Insert(convId string, value []xllm.Message) error
	// 创建会话
	Create(convId string, uid string, title string) error
}
