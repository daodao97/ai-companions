package xllm

import (
	"context"
	"errors"
	"strings"

	"github.com/daodao97/xgo/xlog"
	"github.com/daodao97/xgo/xrequest"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type OpenAIOption func(*OpenAI)

func WithModel(model string) OpenAIOption {
	return func(o *OpenAI) {
		o.model = model
	}
}

func WithAPIUrl(apiUrl string) OpenAIOption {
	return func(o *OpenAI) {
		o.apiUrl = apiUrl
	}
}

func WithAPIKey(apiKey string) OpenAIOption {
	return func(o *OpenAI) {
		o.apiKey = apiKey
	}
}

func WithNotParseToolCall(notParseToolCall bool) OpenAIOption {
	return func(o *OpenAI) {
		o.notParseToolCall = notParseToolCall
	}
}

type OpenAI struct {
	model            string
	apiKey           string
	apiUrl           string
	notParseToolCall bool
}

func NewOpenAI(opts ...OpenAIOption) LLM {
	openai := &OpenAI{}
	for _, opt := range opts {
		opt(openai)
	}
	return openai
}

func (o *OpenAI) request(ctx context.Context, req *xrequest.Request) (*Response, error) {
	request, err := req.
		SetDebug(false).
		SetHeader("Authorization", "Bearer "+o.apiKey).
		SetHeader("Content-Type", "application/json").
		Post(o.apiUrl + "/chat/completions")

	if err != nil {
		return nil, err
	}

	if err := request.Error(); err != nil {
		return nil, err
	}

	response := request.Json()

	result := &Response{
		Content: response.Get("choices.0.message.content").String(),
		Usage: &Usage{
			PromptTokens:     int(response.Get("usage.prompt_tokens").Int()),
			CompletionTokens: int(response.Get("usage.completion_tokens").Int()),
			TotalTokens:      int(response.Get("usage.total_tokens").Int()),
		},
	}

	// 处理工具调用
	if toolCalls := response.Get("choices.0.message.tool_calls"); toolCalls.Exists() {
		var tools []*ToolCall
		toolCalls.ForEach(func(key, value gjson.Result) bool {
			tools = append(tools, &ToolCall{
				ID:   value.Get("id").String(),
				Type: value.Get("type").String(),
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      value.Get("function.name").String(),
					Arguments: value.Get("function.arguments").String(),
				},
			})
			return true // 继续遍历
		})
		result.ToolCall = tools
	}

	return result, nil
}

func (o *OpenAI) requestStream(ctx context.Context, req *xrequest.Request) (chan *Response, error) {
	request, err := req.
		SetDebug(false).
		SetHeader("Authorization", "Bearer "+o.apiKey).
		SetHeader("Content-Type", "application/json").
		Post(o.apiUrl + "/chat/completions")

	if err != nil {
		return nil, err
	}

	stream, err := request.Stream()
	if err != nil {
		return nil, err
	}

	ch := make(chan *Response)

	sendToolCall := func(toolCalls map[int]*ToolCall, fullTool bool) {
		if len(toolCalls) > 0 {
			var toolCallsList []*ToolCall
			for _, toolCall := range toolCalls {
				toolCallsList = append(toolCallsList, toolCall)
			}
			ch <- &Response{ToolCall: toolCallsList, FullTool: fullTool}
		}
	}

	go func() {
		defer close(ch)
		var toolCalls map[int]*ToolCall = make(map[int]*ToolCall)

		defer func() {
			sendToolCall(toolCalls, true)
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case data, ok := <-stream:
				if !ok {
					return
				}

				// 跳过空行和非 data: 开头的行
				if !strings.HasPrefix(data, "data:") {
					continue
				}

				// 移除 "data:" 前缀
				jsonData := strings.TrimSpace(strings.TrimPrefix(data, "data:"))

				if jsonData == "[DONE]" {
					return
				}

				// 解析 JSON 响应
				result := gjson.Parse(jsonData)

				if usage := result.Get("usage"); usage.Exists() && usage.String() != "" && len(usage.Map()) > 0 {
					usageInfo := &Usage{
						PromptTokens:     cast.ToInt(usage.Get("prompt_tokens").Int()),
						CompletionTokens: cast.ToInt(usage.Get("completion_tokens").Int()),
						TotalTokens:      cast.ToInt(usage.Get("total_tokens").Int()),
						CachedTokens:     cast.ToInt(usage.Get("prompt_tokens_details.cached_tokens").Int()),
					}

					if cachedTokens := usage.Get("prompt_tokens_details.cached_tokens"); cachedTokens.Exists() {
						usageInfo.CachedTokens = int(cachedTokens.Int())
					}

					if reasoningTokens := usage.Get("completion_tokens_details.reasoning_tokens"); reasoningTokens.Exists() && reasoningTokens.Int() > 0 {
						usageInfo.CompletionTokens = int(reasoningTokens.Int())
					}

					if usageInfo.PromptTokens != 0 || usageInfo.CompletionTokens != 0 || usageInfo.TotalTokens != 0 {
						ch <- &Response{Usage: usageInfo}
					}

				}

				if result.Get("choices.0.delta.content").Exists() && result.Get("choices.0.delta.content").String() != "" {
					ch <- &Response{Content: result.Get("choices.0.delta.content").String()}
				}

				// 处理工具调用
				if deltaToolCalls := result.Get("choices.0.delta.tool_calls"); deltaToolCalls.Exists() {
					for _, toolCall := range deltaToolCalls.Array() {
						index := int(toolCall.Get("index").Int())

						// 如果是新的工具调用，初始化
						if toolCall.Get("id").Exists() {
							toolCalls[index] = &ToolCall{
								ID:   toolCall.Get("id").String(),
								Type: toolCall.Get("type").String(),
							}
							toolCalls[index].Function.Name = toolCall.Get("function.name").String()
							toolCalls[index].Function.Arguments = ""
						}

						// 累积参数
						if existingToolCall, exists := toolCalls[index]; exists {
							if args := toolCall.Get("function.arguments"); args.Exists() {
								existingToolCall.Function.Arguments += args.String()
							}
						} else {
							xlog.WarnC(ctx, "警告: 工具调用索引 %d 不存在，无法累积参数", index)
						}
					}
					sendToolCall(toolCalls, false)
				}
			}
		}
	}()

	return ch, nil
}

func (o *OpenAI) Chat(ctx context.Context, req Request) (*Response, error) {
	model := o.model
	if req.Model != "" {
		model = req.Model
	}

	body := map[string]any{
		"model":    model,
		"messages": req.Messages,
		"stream_options": map[string]any{
			"include_usage": true,
		},
	}
	if req.ToolChoice != nil {
		body["tool_choice"] = req.ToolChoice
	}
	if len(req.Tools) > 0 {
		body["tools"] = req.Tools
	}

	_req := xrequest.New().SetBody(body)
	return o.request(ctx, _req)
}

func (o *OpenAI) ChatStream(ctx context.Context, req Request) (chan *Response, error) {
	if len(req.Messages) == 0 {
		return nil, errors.New("messages is required")
	}

	model := o.model
	if req.Model != "" {
		model = req.Model
	}

	body := map[string]any{
		"model":    model,
		"messages": req.Messages,
		"stream":   true,
		"stream_options": map[string]any{
			"include_usage": true,
		},
	}
	if len(req.Tools) > 0 {
		body["tools"] = req.Tools
	}
	if req.ToolChoice != nil {
		body["tool_choice"] = req.ToolChoice
	}

	_req := xrequest.New().SetBody(body)
	return o.requestStream(ctx, _req)
}

func (o *OpenAI) ChatRaw(ctx context.Context, body []byte) (*Response, error) {
	_req := xrequest.New().SetBody(body)
	return o.request(ctx, _req)
}

func (o *OpenAI) ChatStreamRaw(ctx context.Context, body []byte) (chan *Response, error) {
	body, err := sjson.SetBytes(body, "stream_options.include_usage", true)
	if err != nil {
		return nil, err
	}
	_req := xrequest.New().SetBody(body)
	return o.requestStream(ctx, _req)
}
