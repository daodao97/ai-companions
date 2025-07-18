package xllm

import (
	"encoding/json"
	"fmt"
	"testing"
)

func Example_textMessage() {
	// 创建纯文本消息
	msg := Message{
		Role:    "user",
		Content: NewTextContent("Hello!"),
	}

	// 序列化为 JSON
	data, _ := json.Marshal(msg)
	fmt.Printf("文本消息: %s\n", string(data))

	// 反序列化
	var parsed Message
	json.Unmarshal(data, &parsed)

	if textContent, ok := parsed.Content.(*TextContent); ok {
		fmt.Printf("解析后的文本: %s\n", textContent.Text)
	}

	// Output:
	// 文本消息: {"role":"user","content":"Hello!"}
	// 解析后的文本: Hello!
}

func Example_multiMediaMessage() {
	// 创建多媒体消息
	items := []ContentItem{
		{
			Type: "text",
			Text: "What is in this image?",
		},
		{
			Type: "image_url",
			ImageURL: &ImageURL{
				URL:    "https://upload.wikimedia.org/wikipedia/commons/thumb/d/dd/Gfp-wisconsin-madison-the-nature-boardwalk.jpg/2560px-Gfp-wisconsin-madison-the-nature-boardwalk.jpg",
				Detail: "high",
			},
		},
	}

	msg := Message{
		Role:    "user",
		Content: NewMultiContent(items),
	}

	// 序列化为 JSON
	data, _ := json.Marshal(msg)
	fmt.Printf("多媒体消息类型: %s\n", msg.Content.Type())

	// 反序列化
	var parsed Message
	json.Unmarshal(data, &parsed)

	if multiContent, ok := parsed.Content.(*MultiContent); ok {
		fmt.Printf("解析后的内容项数量: %d\n", len(multiContent.Items))
		for i, item := range multiContent.Items {
			fmt.Printf("项目 %d: 类型=%s\n", i+1, item.Type)
		}
	}
}

func TestContent_InterfaceDesign(t *testing.T) {
	// 测试接口设计的灵活性

	// 测试文本内容
	textContent := NewTextContent("Hello World!")
	if textContent.Type() != "text" {
		t.Errorf("文本内容类型应该是 'text'，实际得到 '%s'", textContent.Type())
	}

	// 测试多媒体内容
	items := []ContentItem{
		{Type: "text", Text: "描述"},
		{Type: "image_url", ImageURL: &ImageURL{URL: "http://example.com/image.jpg"}},
	}
	multiContent := NewMultiContent(items)
	if multiContent.Type() != "multi" {
		t.Errorf("多媒体内容类型应该是 'multi'，实际得到 '%s'", multiContent.Type())
	}

	// 测试接口多态性
	var contents []Content = []Content{
		textContent,
		multiContent,
	}

	expectedTypes := []string{"text", "multi"}
	for i, content := range contents {
		if content.Type() != expectedTypes[i] {
			t.Errorf("内容 %d 的类型应该是 '%s'，实际得到 '%s'", i, expectedTypes[i], content.Type())
		}
	}
}

func TestContent_JSONCompatibility(t *testing.T) {
	// 测试与 OpenAI API 兼容的 JSON 格式

	// 测试文本格式
	textJSON := `{"role":"user","content":"Hello!"}`
	var textMsg Message
	if err := json.Unmarshal([]byte(textJSON), &textMsg); err != nil {
		t.Errorf("解析文本消息失败: %v", err)
	}

	textContent, ok := textMsg.Content.(*TextContent)
	if !ok {
		t.Errorf("内容应该是 TextContent 类型")
	}
	if textContent.Text != "Hello!" {
		t.Errorf("文本内容错误，期望 'Hello!'，实际得到 '%s'", textContent.Text)
	}

	// 测试多媒体格式
	multiJSON := `{
		"role": "user",
		"content": [
			{
				"type": "text",
				"text": "What is in this image?"
			},
			{
				"type": "image_url",
				"image_url": {
					"url": "https://example.com/image.jpg"
				}
			}
		]
	}`

	var multiMsg Message
	if err := json.Unmarshal([]byte(multiJSON), &multiMsg); err != nil {
		t.Errorf("解析多媒体消息失败: %v", err)
	}

	multiContent, ok := multiMsg.Content.(*MultiContent)
	if !ok {
		t.Errorf("内容应该是 MultiContent 类型")
	}

	if len(multiContent.Items) != 2 {
		t.Errorf("期望 2 个内容项，实际得到 %d 个", len(multiContent.Items))
	}

	if multiContent.Items[0].Type != "text" || multiContent.Items[0].Text != "What is in this image?" {
		t.Errorf("第一个内容项解析错误")
	}

	if multiContent.Items[1].Type != "image_url" || multiContent.Items[1].ImageURL.URL != "https://example.com/image.jpg" {
		t.Errorf("第二个内容项解析错误")
	}
}

func TestContent_Serialization(t *testing.T) {
	// 测试序列化和反序列化的完整性

	// 创建原始消息
	original := Message{
		Role: "user",
		Content: NewMultiContent([]ContentItem{
			{Type: "text", Text: "测试消息"},
			{Type: "image_url", ImageURL: &ImageURL{URL: "http://test.com/img.jpg", Detail: "high"}},
		}),
	}

	// 序列化
	data, err := json.Marshal(original)
	if err != nil {
		t.Errorf("序列化失败: %v", err)
	}

	// 反序列化
	var parsed Message
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("反序列化失败: %v", err)
	}

	// 验证内容
	originalMulti := original.Content.(*MultiContent)
	parsedMulti := parsed.Content.(*MultiContent)

	if len(originalMulti.Items) != len(parsedMulti.Items) {
		t.Errorf("内容项数量不匹配")
	}

	for i := range originalMulti.Items {
		if originalMulti.Items[i].Type != parsedMulti.Items[i].Type {
			t.Errorf("内容项 %d 类型不匹配", i)
		}
		if originalMulti.Items[i].Text != parsedMulti.Items[i].Text {
			t.Errorf("内容项 %d 文本不匹配", i)
		}
	}
}
