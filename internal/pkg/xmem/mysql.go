package xmem

import (
	"companions/internal/conf"
	"companions/internal/pkg/xagent"
	"companions/internal/pkg/xllm"
	"encoding/json"
	"time"

	"github.com/daodao97/xgo/xdb"
	"github.com/tidwall/gjson"
)

type MysqlMemory struct {
	conv xdb.Model
	msg  xdb.Model
}

func NewMysqlMemory(convModel, msgModel xdb.Model) *MysqlMemory {
	return &MysqlMemory{
		conv: convModel,
		msg:  msgModel,
	}
}

func (m *MysqlMemory) Get(key string) (*Conversation, error) {
	conv, err := m.conv.First(xdb.WhereEq("conversation_id", key))
	if err != nil {
		return nil, err
	}

	data := &Conversation{
		Id:        conv.GetString("conversation_id"),
		Title:     conv.GetString("title"),
		Model:     conv.GetString("model"),
		CreatedAt: conv.GetTime("created_at").Format(time.DateTime),
		UpdatedAt: conv.GetTime("updated_at").Format(time.DateTime),
	}

	var messages []any
	list, err := m.msg.Selects(xdb.WhereEq("conversation_id", key), xdb.OrderByAsc("id"))
	if err != nil {
		return nil, err
	}

	// 按 message_id 分组，同时保持顺序
	messageGroups := make(map[string][]map[string]any)
	messageOrder := make([]string, 0) // 记录 message_id 的出现顺序
	seenMessageIds := make(map[string]bool)

	for _, item := range list {
		if item.GetString("message_id") == "summary" {
			continue
		}

		var msg map[string]any
		err := json.Unmarshal([]byte(item.GetString("content")), &msg)
		if err != nil {
			return nil, err
		}
		if msg["role"] == "usage" {
			continue
		}
		msg["message_id"] = item.GetString("message_id")
		delete(msg, "memory")
		delete(msg, "storage")
		delete(msg, "hidden")

		messageId := item.GetString("message_id")

		// 记录第一次出现的 message_id 顺序
		if !seenMessageIds[messageId] {
			messageOrder = append(messageOrder, messageId)
			seenMessageIds[messageId] = true
		}

		messageGroups[messageId] = append(messageGroups[messageId], msg)
	}

	// 按原顺序将分组后的消息添加到结果中
	for _, messageId := range messageOrder {
		group := messageGroups[messageId]
		messages = append(messages, group)
	}

	data.Messages = messages

	return data, nil
}

func (m *MysqlMemory) Create(convId string, uid string, title string) error {
	_, err := m.conv.Insert(xdb.Record{
		"conversation_id": convId,
		"uid":             uid,
		"title":           title,
	})
	if err != nil {
		return err
	}
	return nil
}

func (m *MysqlMemory) Insert(key string, value []xagent.Message) error {
	var records []xdb.Record
	processor := xagent.NewMessageProcessor()

	for _, item := range value {
		records = append(records, xdb.Record{
			"conversation_id": key,
			"message_id":      item.GetMessageID(),
			"content":         processor.ToJSON(item),
		})
	}

	if len(records) > 0 {
		_, err := m.msg.InsertBatch(records)
		return err
	}

	return nil
}

// 获取记忆
// 检查历史消息是否超出 模型最大token数
// 如果超出，则压缩历史消息
// 压缩历史消息时，使用 compressPrompt 提示词
func (m *MysqlMemory) GetMemory(convId string) ([]xllm.Message, error) {
	var messages []xllm.Message
	list, err := m.msg.Selects(xdb.WhereEq("conversation_id", convId), xdb.OrderByAsc("created_at"))
	if err != nil {
		return nil, err
	}

	// TODO: RAG 过滤

	var lastUsageMessage string
	for _, item := range list {
		role := gjson.Get(item.GetString("content"), "role").String()
		step := gjson.Get(item.GetString("content"), "step").String()
		if role == "usage" && step == "llm_call" {
			lastUsageMessage = item.GetString("content")
			continue
		}

		isSummary := gjson.Get(item.GetString("content"), "summary").Bool()
		if isSummary {
			llmMsg := ToLLmMessage(item.GetString("content"))
			if llmMsg != nil {
				messages = []xllm.Message{*llmMsg}
			}
			lastUsageMessage = ""
			continue
		}

		llmMsg := ToLLmMessage(item.GetString("content"))
		if llmMsg != nil {
			messages = append(messages, *llmMsg)
		}

	}

	// 如果最后一条消息是 summary 消息，则不进行压缩
	if lastUsageMessage == "" {
		return messages, nil
	}

	totalTokens := gjson.Get(lastUsageMessage, "usage.total_tokens").Int()
	limit := GetModelMaxTokens(gjson.Get(lastUsageMessage, "model").String())

	if totalTokens >= int64(float64(limit)*0.8) {
		compressMessages := Compress(messages)

		// 创建压缩后的 LLM 消息
		messages = []xllm.Message{
			{
				Role:    xllm.RoleAssistant,
				Content: xllm.NewTextContent(compressMessages),
			},
		}

		// 使用新的转换函数创建 agent 消息
		agentMessages := xagent.ToAgentMessages(messages)
		m.Insert(convId, agentMessages)
	}

	return messages, nil
}

func GetModelMaxTokens(model string) int {
	models := conf.Get().LLM
	for _, item := range models {
		if item.Model == model {
			if item.MaxTokens > 0 {
				return item.MaxTokens
			}
			return 100000
		}
	}
	return 100000
}

func ToLLmMessage(msgStr string) *xllm.Message {
	// 解析 JSON 字符串为 agent.Message
	var baseMsg xagent.BaseMessage
	if err := json.Unmarshal([]byte(msgStr), &baseMsg); err != nil {
		return nil
	}

	// 使用新的转换函数
	agentMsg := &baseMsg
	llmMsg := xagent.ToLLMMessage(agentMsg)
	return &llmMsg
}
