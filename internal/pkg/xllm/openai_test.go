package xllm

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestOpenAI_Chat(t *testing.T) {
	openai := NewOpenAI(
		WithModel("gpt-4o-mini"),
		WithAPIUrl(os.Getenv("OPENAI_BASE_URL")),
		WithAPIKey(os.Getenv("OPENAI_API_KEY")),
	)

	response, err := openai.Chat(context.Background(), Request{
		Model:    "gpt-4o-mini",
		Messages: []Message{{Role: "user", Content: NewTextContent("hello")}},
	})
	if err != nil {
		t.Errorf("Chat 失败: %+v\n", err)
	}
	spew.Dump(response)
}

func TestOpenAI_ChatStream(t *testing.T) {
	openai := NewOpenAI(
		WithModel("gpt-4o-mini"),
		WithAPIUrl(os.Getenv("OPENAI_BASE_URL")),
		WithAPIKey(os.Getenv("OPENAI_API_KEY")),
	)
	stream, err := openai.ChatStream(context.Background(), Request{
		Model:    "gpt-4o-mini",
		Messages: []Message{{Role: "user", Content: NewTextContent("上海天气如何")}},
		Tools: []Tool{
			{
				Name:        "get_current_weather",
				Description: "Get the current weather in a given location",
				Parameters: []Parameter{
					{Name: "location", Description: "The location to get the weather for", Type: "string", Required: true},
				},
			},
		},
	})
	if err != nil {
		t.Errorf("ChatStream 失败: %v", err)
	}

	for data := range stream {
		if data.Content != "" {
			fmt.Println(data.Content)
		}
		if data.ToolCall != nil {
			fmt.Printf("tool call: %+v\n", data.ToolCall)
		}
		if data.Usage != nil {
			fmt.Printf("usage: %+v\n", data.Usage)
		}
	}

}
