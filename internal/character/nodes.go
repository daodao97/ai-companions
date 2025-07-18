package character

import (
	"companions/internal/pkg/xagent"
	"companions/internal/pkg/xflow"
	"companions/internal/pkg/xllm"
	"companions/internal/pkg/xmem"
	"companions/internal/pkg/xtools"
	"companions/internal/pkg/xtts"
	"context"
	"strconv"
	"strings"

	"github.com/daodao97/xgo/xlog"
)

// AI伴侣工作流状态 - 基于实际流程
type AICompanionState struct {
	// 输入
	UserMessage string
	UserID      string

	// LLM相关
	LLMResponse string
	LLM         xllm.LLM

	// 记忆相关
	Memory  *xmem.MysqlMemory
	History []xllm.Message

	// TTS相关
	TTS       xtts.TTS
	MessageID string

	// 输出消息流
	MessageStream chan xagent.Message

	// 扩展功能
	RomanceMeter int
	ActionTaken  string
}

// LLM聊天响应和TTS生成节点 - 合并为一个节点
type LLMChatAndTTSNode struct {
	xflow.BaseNode
}

func NewLLMChatAndTTSNode() *LLMChatAndTTSNode {
	return &LLMChatAndTTSNode{
		BaseNode: xflow.BaseNode{
			Name: "llm_chat_and_tts",
			Type: xflow.NodeTypeExecute,
		},
	}
}

func (l *LLMChatAndTTSNode) Execute(ctx context.Context, state *AICompanionState) (*xflow.NodeResult[AICompanionState], error) {
	allMsg := append([]xllm.Message{
		{Role: "assistant", Content: xllm.NewTextContent(Ani.Instructions)},
	}, state.History...)
	allMsg = append(allMsg, xllm.Message{
		Role:    "user",
		Content: xllm.NewTextContent(state.UserMessage),
	})

	state.History = allMsg

	// 调用LLM
	chatResp, err := state.LLM.Chat(ctx, xllm.Request{
		Messages: allMsg,
	})
	if err != nil {
		xlog.ErrorC(ctx, "LLM请求失败", xlog.Err(err))
		return &xflow.NodeResult[AICompanionState]{
			Success: false,
			Error:   err,
			State:   state,
		}, nil
	}

	state.LLMResponse = chatResp.Content

	// 保存到记忆
	if state.UserID != "" {
		agentMessages := []xagent.Message{
			xagent.NewUserMessage(state.UserMessage),
			xagent.NewAssistantMessage(chatResp.Content),
		}
		state.Memory.Insert(state.UserID, agentMessages)
		state.History = append(state.History, xllm.Message{
			Role:    xllm.RoleAssistant,
			Content: xllm.NewTextContent(chatResp.Content),
		})
	}

	// 发送文本响应消息到流
	if state.MessageStream != nil {
		textMsg := xagent.NewAssistantMessage(chatResp.Content)
		state.MessageStream <- textMsg
	}

	// 生成音频流
	audioStream, err := state.TTS.TextToSpeech(xtts.AudioReq{
		Text: state.LLMResponse,
	})
	if err != nil {
		xlog.ErrorC(ctx, "TTS请求失败", xlog.Err(err))
		return &xflow.NodeResult[AICompanionState]{
			Success: false,
			Error:   err,
			State:   state,
		}, nil
	}

	for chunk := range audioStream {
		state.MessageStream <- NewAudioMessage(chunk)
	}

	xlog.InfoC(ctx, "LLM响应和TTS生成完成", xlog.String("content", state.LLMResponse), xlog.String("message_id", state.MessageID))

	return &xflow.NodeResult[AICompanionState]{
		Success: true,
		State:   state,
	}, nil
}

// 浪漫度变化节点 - 扩展功能
type RomanceMeterChangeNode struct {
	xflow.BaseNode
}

func NewRomanceMeterChangeNode() *RomanceMeterChangeNode {
	return &RomanceMeterChangeNode{
		BaseNode: xflow.BaseNode{
			Name: "romance_meter_change",
			Type: xflow.NodeTypeExecute,
		},
	}
}

func (r *RomanceMeterChangeNode) Execute(ctx context.Context, state *AICompanionState) (*xflow.NodeResult[AICompanionState], error) {
	chatResp, err := state.LLM.Chat(ctx, xllm.Request{
		Messages: []xllm.Message{
			{
				Role:    xllm.RoleAssistant,
				Content: xllm.NewTextContent(Ani.GetRomancePrompt(state.History)),
			},
			{
				Role:    xllm.RoleUser,
				Content: xllm.NewTextContent(state.UserMessage),
			},
		},
	})

	if err != nil {
		return &xflow.NodeResult[AICompanionState]{
			Success: false,
			Error:   err,
			State:   state,
		}, nil
	}

	var change int

	if stateResult, err := xtools.XmlAttr(chatResp.Content, "romance_meter_change"); err == nil {
		changeStr := strings.TrimPrefix(stateResult.String(), "+")
		if parsedChange, parseErr := strconv.Atoi(changeStr); parseErr == nil {
			change = parsedChange
		}
		xlog.InfoC(ctx, "浪漫度变化", xlog.Int("change", change))
	}

	state.RomanceMeter += change

	if state.MessageStream != nil {
		romanceMsg := NewRomanceMessage(state.RomanceMeter)
		state.MessageStream <- romanceMsg
	}

	xlog.InfoC(ctx, "浪漫度更新", xlog.Int("change", change), xlog.Int("new_value", state.RomanceMeter))

	return &xflow.NodeResult[AICompanionState]{
		Success: true,
		State:   state,
	}, nil
}

// 动作执行节点 - 扩展功能
type ActionNode struct {
	xflow.BaseNode
}

func NewActionNode() *ActionNode {
	return &ActionNode{
		BaseNode: xflow.BaseNode{
			Name: "action",
			Type: xflow.NodeTypeExecute,
		},
	}
}

func (a *ActionNode) Execute(ctx context.Context, state *AICompanionState) (*xflow.NodeResult[AICompanionState], error) {
	allMsg := append([]xllm.Message{
		{Role: "assistant", Content: xllm.NewTextContent(Ani.ActionPrompt)},
	}, state.History...)
	allMsg = append(allMsg, xllm.Message{
		Role:    xllm.RoleUser,
		Content: xllm.NewTextContent(state.UserMessage),
	})

	chatResp, err := state.LLM.Chat(ctx, xllm.Request{
		Messages: allMsg,
		Tools: []xllm.Tool{
			{
				Name:        "heartbeat",
				Description: "Directs the avatar to perform a heartbeat animation.",
				Parameters:  []xllm.Parameter{},
			},
			{
				Name:        "updateMusicState",
				Description: "Directs the avatar to change clothes based on vocal commands.",
				Parameters: []xllm.Parameter{
					{
						Name:        "music_state",
						Type:        "string",
						Description: "sets the state of the music",
						Required:    true,
						Enum:        []string{"play", "stop", "switch_track"},
					},
				},
			},
			{
				Name:        "showEmotion",
				Description: "Directs the avatar to display emotions: curiosity, shyness, stress, sadness, frustration.",
				Parameters: []xllm.Parameter{
					{
						Name:        "emotion",
						Type:        "string",
						Description: "Emotion to display",
						Required:    true,
						Enum:        []string{"curiosity", "shyness", "stress", "sadness", "frustration"},
					},
				},
			},
			{
				Name:        "move",
				Description: "Directs the avatar to move based on vocal commands.",
				Parameters: []xllm.Parameter{
					{
						Name:        "action",
						Type:        "string",
						Description: "Action to perform",
						Required:    true,
						Enum:        []string{"spin_1", "peek", "sway_1", "sway_2", "tease", "kiss"},
					},
					{
						Name:        "repeat_count",
						Type:        "string",
						Description: "Number of times to repeat the action",
						Required:    true,
						Enum:        []string{"1", "2", "3", "4"},
					},
				},
			},
			{
				Name:        "showAllMoves",
				Description: "Directs the avatar to show all moves.",
				Parameters:  []xllm.Parameter{},
			},
			{
				Name:        "stopMove",
				Description: "Directs the avatar to stop any moves.",
				Parameters:  []xllm.Parameter{},
			},
			{
				Name:        "hideBackground",
				Description: "Directs the avatar to hide background.",
				Parameters:  []xllm.Parameter{},
			},
			// dress up and undress
			{
				Name:        "dressUp",
				Description: "Dress up the avatar",
				Parameters: []xllm.Parameter{
					{
						Name:        "clothes",
						Type:        "string",
						Description: "The clothes to wear",
					},
				},
			},
			{
				Name:        "undress",
				Description: "Undress the avatar",
				Parameters: []xllm.Parameter{
					{
						Name:        "clothes",
						Type:        "string",
						Description: "The clothes to undress",
					},
				},
			},
		},
	})

	if err != nil {
		return &xflow.NodeResult[AICompanionState]{
			Success: false,
			Error:   err,
			State:   state,
		}, nil
	}

	if len(chatResp.ToolCall) == 0 {
		return &xflow.NodeResult[AICompanionState]{
			Success: true,
			State:   state,
		}, nil
	}

	// 发送动作消息
	if state.MessageStream != nil {
		actionMsg := NewActionMessage(chatResp.ToolCall[0])
		state.MessageStream <- actionMsg
	}

	xlog.InfoC(ctx, "执行动作", xlog.Any("action", chatResp.ToolCall))

	return &xflow.NodeResult[AICompanionState]{
		Success: true,
		State:   state,
	}, nil
}
