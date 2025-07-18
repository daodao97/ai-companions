package character

import (
	"companions/internal/pkg/xagent"
	"companions/internal/pkg/xllm"
	"companions/internal/pkg/xtts"
)

// AudioMessage 音频消息
type AudioMessage struct {
	xagent.BaseMessage
	xtts.AudioChunk
}

func NewAudioMessage(chunk xtts.AudioChunk) *AudioMessage {
	return &AudioMessage{
		BaseMessage: xagent.BaseMessage{
			Role: xagent.MessageRoleAssistant,
			Metadata: xagent.MessageMetadata{
				Memory:  false,
				Storage: false,
			},
		},
		AudioChunk: chunk,
	}
}

// ErrorMessage 错误消息
type ErrorMessage struct {
	xagent.BaseMessage
	Error string
}

func NewErrorMessage(error string) *ErrorMessage {
	return &ErrorMessage{
		BaseMessage: xagent.BaseMessage{
			Role: xagent.MessageRoleSystem,
			Metadata: xagent.MessageMetadata{
				Memory:  false,
				Storage: false,
			},
		},
		Error: error,
	}
}

// RomanceMessage 浪漫度消息
type RomanceMessage struct {
	xagent.BaseMessage
	Romance int
}

func NewRomanceMessage(romance int) *RomanceMessage {
	return &RomanceMessage{
		BaseMessage: xagent.BaseMessage{
			Role: xagent.MessageRoleAssistant,
			Metadata: xagent.MessageMetadata{
				Memory:  false,
				Storage: false,
			},
		},
		Romance: romance,
	}
}

// ActionMessage 动作消息
type ActionMessage struct {
	xagent.BaseMessage
	Action string `json:"action"`
	Args   string `json:"args"`
}

func NewActionMessage(action *xllm.ToolCall) *ActionMessage {
	return &ActionMessage{
		BaseMessage: xagent.BaseMessage{
			Role: xagent.MessageRoleAssistant,
			Metadata: xagent.MessageMetadata{
				Memory:  false,
				Storage: false,
			},
		},
		Action: action.Function.Name,
		Args:   action.Function.Arguments,
	}
}
