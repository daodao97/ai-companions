package xagent

import (
	"companions/internal/pkg/xllm"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// MessageRole 消息角色枚举
type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleSystem    MessageRole = "system"
	MessageRoleTool      MessageRole = "tool"
	MessageRoleError     MessageRole = "error"
)

// Message 核心消息接口 - 只关注消息的核心属性
type Message interface {
	GetRole() MessageRole
	GetContent() string
	GetTimestamp() time.Time
	GetMessageID() string
}

// MessageMetadata 消息元数据 - 控制消息的行为
type MessageMetadata struct {
	Memory  bool `json:"memory,omitempty"`  // 是否进入记忆
	Hidden  bool `json:"hidden,omitempty"`  // 是否隐藏（不发送给客户端）
	Storage bool `json:"storage,omitempty"` // 是否存储到数据库
}

// BaseMessage 基础消息实现
type BaseMessage struct {
	Role      MessageRole     `json:"role"`
	Content   string          `json:"content"`
	Timestamp time.Time       `json:"timestamp"`
	MessageID string          `json:"message_id"`
	Metadata  MessageMetadata `json:"metadata,omitempty"`
}

func (m *BaseMessage) GetRole() MessageRole    { return m.Role }
func (m *BaseMessage) GetContent() string      { return m.Content }
func (m *BaseMessage) GetTimestamp() time.Time { return m.Timestamp }
func (m *BaseMessage) GetMessageID() string    { return m.MessageID }

// MessageBuilder 消息构建器 - 提供流畅的API
type MessageBuilder struct {
	msg *BaseMessage
}

func NewMessage() *MessageBuilder {
	return &MessageBuilder{
		msg: &BaseMessage{
			Timestamp: time.Now(),
			MessageID: generateMessageID(),
		},
	}
}

func (b *MessageBuilder) Role(msgRole MessageRole) *MessageBuilder {
	b.msg.Role = msgRole
	return b
}

func (b *MessageBuilder) Content(content string) *MessageBuilder {
	b.msg.Content = content
	return b
}

func (b *MessageBuilder) Memory(memory bool) *MessageBuilder {
	b.msg.Metadata.Memory = memory
	return b
}

func (b *MessageBuilder) Hidden(hidden bool) *MessageBuilder {
	b.msg.Metadata.Hidden = hidden
	return b
}

func (b *MessageBuilder) Storage(storage bool) *MessageBuilder {
	b.msg.Metadata.Storage = storage
	return b
}

func (b *MessageBuilder) Build() Message {
	return b.msg
}

// MessageProcessor 消息处理器 - 处理消息的各种操作
type MessageProcessor struct{}

func NewMessageProcessor() *MessageProcessor {
	return &MessageProcessor{}
}

// ToJSON 序列化消息
func (p *MessageProcessor) ToJSON(msg Message) string {
	data, _ := json.Marshal(msg)
	return string(data)
}

// ToSSE 转换为SSE格式
func (p *MessageProcessor) ToSSE(msg Message) string {
	return fmt.Sprintf("event: %s\ndata: %s\n\n", msg.GetRole(), p.ToJSON(msg))
}

// ShouldStore 判断是否应该存储
func (p *MessageProcessor) ShouldStore(msg Message) bool {
	if base, ok := msg.(*BaseMessage); ok {
		return base.Metadata.Storage
	}
	return true // 默认存储
}

// ShouldSend 判断是否应该发送给客户端
func (p *MessageProcessor) ShouldSend(msg Message) bool {
	if base, ok := msg.(*BaseMessage); ok {
		return !base.Metadata.Hidden
	}
	return true // 默认发送
}

// 便捷创建函数
func NewUserMessage(content string) Message {
	return NewMessage().
		Role(MessageRoleUser).
		Content(content).
		Storage(true).
		Build()
}

func NewAssistantMessage(content string) Message {
	return NewMessage().
		Role(MessageRoleAssistant).
		Content(content).
		Storage(true).
		Build()
}

func NewSystemMessage(content string) Message {
	return NewMessage().
		Role(MessageRoleSystem).
		Content(content).
		Memory(true).
		Hidden(true).
		Build()
}

func NewToolMessage(content string) Message {
	return NewMessage().
		Role(MessageRoleTool).
		Content(content).
		Storage(false).
		Build()
}

func NewErrorMessage(content string) Message {
	return NewMessage().
		Role(MessageRoleError).
		Content(content).
		Storage(false).
		Build()
}

// 工具函数
func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

// 从 Content 接口提取文本内容
func extractContentText(content xllm.Content) string {
	if content == nil {
		return ""
	}

	switch c := content.(type) {
	case *xllm.TextContent:
		return c.Text
	case *xllm.MultiContent:
		var texts []string
		for _, item := range c.Items {
			if item.Type == "text" && item.Text != "" {
				texts = append(texts, item.Text)
			}
		}
		return strings.Join(texts, "\n")
	default:
		// 对于其他类型，尝试序列化为JSON
		if data, err := json.Marshal(content); err == nil {
			return string(data)
		}
		return ""
	}
}

// xllm.Message => agent.Message
func ToAgentMessage(msg xllm.Message) Message {
	// 根据角色映射到对应的 MessageRole
	var role MessageRole
	switch msg.Role {
	case xllm.RoleUser:
		role = MessageRoleUser
	case xllm.RoleAssistant:
		role = MessageRoleAssistant
	case xllm.RoleSystem:
		role = MessageRoleSystem
	case xllm.RoleTool:
		role = MessageRoleTool
	default:
		role = MessageRoleUser // 默认用户角色
	}

	// 提取内容文本
	content := extractContentText(msg.Content)

	// 构建消息
	builder := NewMessage().
		Role(role).
		Content(content)

	// 根据角色设置元数据
	switch role {
	case MessageRoleSystem:
		builder = builder.Memory(true).Hidden(true)
	case MessageRoleTool:
		builder = builder.Storage(false)
	case MessageRoleUser, MessageRoleAssistant:
		builder = builder.Storage(true)
	}

	return builder.Build()
}

// ToAgentMessages 批量转换 xllm.Message 到 agent.Message
func ToAgentMessages(messages []xllm.Message) []Message {
	result := make([]Message, len(messages))
	for i, msg := range messages {
		result[i] = ToAgentMessage(msg)
	}
	return result
}

// ToLLMMessage agent.Message => xllm.Message
func ToLLMMessage(msg Message) xllm.Message {
	// 根据 MessageRole 映射到对应的角色
	var role string
	switch msg.GetRole() {
	case MessageRoleUser:
		role = xllm.RoleUser
	case MessageRoleAssistant:
		role = xllm.RoleAssistant
	case MessageRoleSystem:
		role = xllm.RoleSystem
	case MessageRoleTool:
		role = xllm.RoleTool
	default:
		role = xllm.RoleUser // 默认用户角色
	}

	return xllm.Message{
		Role:    role,
		Content: xllm.NewTextContent(msg.GetContent()),
	}
}

// ToLLMMessages 批量转换 agent.Message 到 xllm.Message
func ToLLMMessages(messages []Message) []xllm.Message {
	result := make([]xllm.Message, len(messages))
	for i, msg := range messages {
		result[i] = ToLLMMessage(msg)
	}
	return result
}
