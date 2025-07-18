package xmem

import (
	"companions/internal/conf"
	"context"

	"companions/internal/pkg/xllm"
)

var compressPrompt = `
Summarize our conversation up to this point. 
The summary should be a concise yet comprehensive overview of all key topics, questions, answers, 
and important details discussed. This summary will replace the current chat history to conserve tokens, 
so it must capture everything essential to understand the context and continue our conversation effectively as if no information was lost.
`

func Compress(messages []xllm.Message) string {
	llmConf := conf.Get().GetLLM("default")
	llm := xllm.NewOpenAI(
		xllm.WithAPIKey(llmConf.ApiKey),
		xllm.WithAPIUrl(llmConf.ApiUrl),
	)
	response, err := llm.Chat(context.Background(), xllm.Request{
		Model: llmConf.Model,
		Messages: append(messages, xllm.Message{
			Role: "user",
			Content: xllm.NewMultiContent([]xllm.ContentItem{
				{
					Type: "text",
					Text: compressPrompt,
				},
			}),
		}),
	})
	if err != nil {
		return ""
	}
	return response.Content
}
