package xllm

import (
	"companions/internal/conf"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Content 表示消息内容的接口
type Content interface {
	// MarshalJSON 实现 JSON 序列化
	MarshalJSON() ([]byte, error)
	// Type 返回内容类型
	Type() string
}

// TextContent 表示纯文本内容
type TextContent struct {
	Text string `json:"-"`
}

// NewTextContent 创建文本内容
func NewTextContent(text string) *TextContent {
	return &TextContent{Text: text}
}

func (t *TextContent) Type() string {
	return "text"
}

func (t *TextContent) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Text)
}

func NewTextContentItem(text string) ContentItem {
	return ContentItem{
		Type: "text",
		Text: text,
	}
}

func NewImageUrlContent(url string) ContentItem {
	return ContentItem{
		Type: "image_url",
		ImageURL: &ImageURL{
			URL: url,
		},
	}
}

// MultiContent 表示多媒体内容
type MultiContent struct {
	Items []ContentItem `json:"-"`
}

// ContentItem 表示内容项
type ContentItem struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL 表示图片URL信息
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // "low", "high", "auto"
}

// NewMultiContent 创建多媒体内容
func NewMultiContent(items []ContentItem) *MultiContent {
	return &MultiContent{Items: items}
}

func (m *MultiContent) Type() string {
	return "multi"
}

func (m *MultiContent) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Items)
}

// ParseContent 解析 JSON 数据为 Content 接口
func ParseContent(data []byte) (Content, error) {
	// 首先尝试解析为字符串
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		return NewTextContent(text), nil
	}

	// 如果不是字符串，尝试解析为 ContentItem 数组
	var items []ContentItem
	if err := json.Unmarshal(data, &items); err == nil {
		return NewMultiContent(items), nil
	}

	return nil, json.Unmarshal(data, &items)
}

const (
	RoleUser      string = "user"
	RoleAssistant string = "assistant"
	RoleSystem    string = "system"
	RoleTool      string = "tool"
	RoleFunction  string = "function"
)

// Message 表示消息
type Message struct {
	Role       string      `json:"role"`
	Content    Content     `json:"content,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
	ToolCalls  []*ToolCall `json:"tool_calls,omitempty"`
	Tools      []Tool      `json:"tools,omitempty"`
}

// UnmarshalJSON 实现 Message 的 JSON 反序列化
func (m *Message) UnmarshalJSON(data []byte) error {
	// 定义临时结构体
	var temp struct {
		Role       string          `json:"role"`
		Content    json.RawMessage `json:"content"`
		ToolCallID string          `json:"tool_call_id,omitempty"`
		ToolCalls  []*ToolCall     `json:"tool_calls,omitempty"`
		Tools      []Tool          `json:"tools,omitempty"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	m.Role = temp.Role
	m.ToolCallID = temp.ToolCallID
	m.ToolCalls = temp.ToolCalls
	m.Tools = temp.Tools

	// 解析 Content
	content, err := ParseContent(temp.Content)
	if err != nil {
		return err
	}
	m.Content = content

	return nil
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  []Parameter `json:"parameters"`
}

type Parameter struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Type        string     `json:"type"` // string, number, integer, boolean, array, object
	Required    bool       `json:"required"`
	Items       *Parameter `json:"items,omitempty"` // 用于数组类型的元素定义
	Enum        []string   `json:"enum,omitempty"`  // 用于枚举类型的定义
}

const (
	ParameterTypeString  = "string"
	ParameterTypeNumber  = "number"
	ParameterTypeInteger = "integer"
	ParameterTypeBoolean = "boolean"
	ParameterTypeArray   = "array"
	ParameterTypeObject  = "object"
)

// MarshalJSON 实现 Tool 的自定义 JSON 序列化
func (t Tool) MarshalJSON() ([]byte, error) {
	// 构建 properties
	properties := make(map[string]any)
	var required []string

	for _, param := range t.Parameters {
		paramSchema := map[string]any{
			"type":        param.Type,
			"description": param.Description,
		}

		// 如果是数组类型且有 items 定义，添加 items 属性
		if param.Type == "array" && param.Items != nil {
			itemsSchema := map[string]any{
				"type": param.Items.Type,
			}
			if param.Items.Description != "" {
				itemsSchema["description"] = param.Items.Description
			}
			paramSchema["items"] = itemsSchema
		}

		properties[param.Name] = paramSchema
		if param.Required {
			required = append(required, param.Name)
		}
		if len(param.Enum) > 0 {
			paramSchema["enum"] = param.Enum
		}
	}

	parameters := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}

	if len(required) > 0 {
		parameters["required"] = required
	}

	function := map[string]any{
		"name":        t.Name,
		"description": t.Description,
		"strict":      true,
	}

	if len(properties) > 0 {
		function["parameters"] = parameters
	}

	openaiTool := map[string]any{
		"type":     "function",
		"function": function,
	}

	return json.Marshal(openaiTool)
}

// ToolChoice 表示工具选择策略
type ToolChoice interface {
	// MarshalJSON 实现 JSON 序列化
	MarshalJSON() ([]byte, error)
	// Type 返回工具选择类型
	Type() string
}

// StringToolChoice 表示字符串类型的工具选择
type StringToolChoice struct {
	Value string
}

// NewStringToolChoice 创建字符串类型的工具选择
func NewStringToolChoice(value string) *StringToolChoice {
	return &StringToolChoice{Value: value}
}

func (s *StringToolChoice) Type() string {
	return "string"
}

func (s *StringToolChoice) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Value)
}

// ObjectToolChoice 表示对象类型的工具选择
type ObjectToolChoice struct {
	ToolType string `json:"type"`
	Function struct {
		Name string `json:"name"`
	} `json:"function"`
}

// NewObjectToolChoice 创建对象类型的工具选择
func NewObjectToolChoice(functionName string) *ObjectToolChoice {
	return &ObjectToolChoice{
		ToolType: "function",
		Function: struct {
			Name string `json:"name"`
		}{Name: functionName},
	}
}

func (o *ObjectToolChoice) Type() string {
	return "object"
}

func (o *ObjectToolChoice) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type     string `json:"type"`
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}{
		Type:     o.ToolType,
		Function: o.Function,
	})
}

// ParseToolChoice 解析 JSON 数据为 ToolChoice 接口
func ParseToolChoice(data []byte) (ToolChoice, error) {
	// 首先尝试解析为字符串
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		return NewStringToolChoice(str), nil
	}

	// 如果不是字符串，尝试解析为对象
	var obj ObjectToolChoice
	if err := json.Unmarshal(data, &obj); err == nil {
		return &obj, nil
	}

	return nil, fmt.Errorf("无法解析 tool_choice 字段")
}

type Request struct {
	Model      string     `json:"model"`
	Messages   []Message  `json:"messages"`
	Tools      []Tool     `json:"tools,omitempty"`
	ToolChoice ToolChoice `json:"tool_choice,omitempty"`
}

// UnmarshalJSON 实现 Request 的 JSON 反序列化
func (r *Request) UnmarshalJSON(data []byte) error {
	// 定义临时结构体
	var temp struct {
		Model      string          `json:"model"`
		Messages   []Message       `json:"messages"`
		Tools      []Tool          `json:"tools,omitempty"`
		ToolChoice json.RawMessage `json:"tool_choice,omitempty"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	r.Model = temp.Model
	r.Messages = temp.Messages
	r.Tools = temp.Tools

	// 解析 ToolChoice
	if len(temp.ToolChoice) > 0 {
		toolChoice, err := ParseToolChoice(temp.ToolChoice)
		if err != nil {
			return err
		}
		r.ToolChoice = toolChoice
	}

	return nil
}

type Response struct {
	Content  string      `json:"content,omitempty"`
	FullTool bool        `json:"full_tool,omitempty"`
	ToolCall []*ToolCall `json:"tool_call,omitempty"`
	Usage    *Usage      `json:"usage,omitempty"`
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	CachedTokens     int `json:"cached_tokens,omitempty"`
}

type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
	AudioTokens  int `json:"audio_tokens"`
}

type CompletionTokensDetails struct {
	ReasoningTokens          int `json:"reasoning_tokens"`
	AudioTokens              int `json:"audio_tokens"`
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens"`
	RejectedPredictionTokens int `json:"rejected_prediction_tokens"`
}

type LLM interface {
	Chat(ctx context.Context, req Request) (*Response, error)
	ChatStream(ctx context.Context, req Request) (chan *Response, error)
	ChatRaw(ctx context.Context, body []byte) (*Response, error)
	ChatStreamRaw(ctx context.Context, body []byte) (chan *Response, error)
}

type FileItem struct {
	Type    string `json:"type"`
	Name    string `json:"name,omitempty"`
	Url     string `json:"url,omitempty"`
	Size    string `json:"size,omitempty"`
	Content string `json:"content,omitempty"`
}

func FileToMessage(txtMsg string, files []FileItem) *MultiContent {
	items := make([]ContentItem, 0)
	if txtMsg != "" {
		items = append(items, NewTextContentItem(txtMsg))
	}
	for _, file := range files {
		fileUrl := file.Url
		if !strings.HasPrefix(fileUrl, "http") {
			fileUrl = "https://chatflex.oss-ap-northeast-1.aliyuncs.com/" + fileUrl
		}
		file.Url = fileUrl
		if strings.HasPrefix(file.Type, "image") {
			items = append(items, NewImageUrlContent(fileUrl))
		}
		items = append(items, NewTextContentItem(fmt.Sprintf("user upload file name: %s\n file type: %s\n file url: %s\n file content: %s", file.Name, file.Type, file.Url, file.Content)))
	}
	return NewMultiContent(items)
}

type LLmMessage interface {
	Messages() []Message
}

func New(llmConf *conf.LLMConfig) LLM {
	switch llmConf.Provider {
	case "openai":
		return NewOpenAI(
			WithAPIKey(llmConf.ApiKey),
			WithAPIUrl(llmConf.ApiUrl),
			WithModel(llmConf.Model),
		)
	}
	return nil
}
