package character

import (
	"testing"
)

func TestXmlAttr(t *testing.T) {
	tests := []struct {
		name     string
		xmlStr   string
		keyPath  string
		expected string
	}{
		{
			name:     "简单标签提取",
			xmlStr:   `<romance_meter_change>5</romance_meter_change>`,
			keyPath:  "romance_meter_change",
			expected: "5",
		},
		{
			name:     "嵌套标签提取",
			xmlStr:   `<root><child><grandchild>value</grandchild></child></root>`,
			keyPath:  "root/child/grandchild",
			expected: "value",
		},
		{
			name:     "带属性的标签提取",
			xmlStr:   `<tag id="123" name="test">content</tag>`,
			keyPath:  "tag@id=123",
			expected: "content",
		},
		{
			name:     "带属性的标签提取 - 使用name属性",
			xmlStr:   `<tag id="123" name="test">content</tag>`,
			keyPath:  "tag@name=test",
			expected: "content",
		},
		{
			name:     "复杂XML结构",
			xmlStr:   `<response><data><romance_meter_change>10</romance_meter_change><status>success</status></data></response>`,
			keyPath:  "response/data/romance_meter_change",
			expected: "10",
		},
		{
			name:     "多个相同标签 - 提取第一个",
			xmlStr:   `<item>first</item><item>second</item>`,
			keyPath:  "item",
			expected: "first",
		},
		{
			name:     "标签不存在",
			xmlStr:   `<root><child>value</child></root>`,
			keyPath:  "nonexistent",
			expected: "",
		},
		{
			name:     "空路径",
			xmlStr:   `<tag>value</tag>`,
			keyPath:  "",
			expected: "",
		},
		{
			name:     "带空格的XML",
			xmlStr:   `  <tag>  value  </tag>  `,
			keyPath:  "tag",
			expected: "value",
		},
		{
			name:     "带换行的XML",
			xmlStr:   `<tag>\n  value\n</tag>`,
			keyPath:  "tag",
			expected: "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := XmlAttr(tt.xmlStr, tt.keyPath)
			if result != tt.expected {
				t.Errorf("XmlAttr(%q, %q) = %q, want %q", tt.xmlStr, tt.keyPath, result, tt.expected)
			}
		})
	}
}

func TestXmlAttrWithRomanceMeter(t *testing.T) {
	// 测试实际的浪漫度变化XML
	xmlStr := `<romance_meter_change>5</romance_meter_change>`

	result := XmlAttr(xmlStr, "romance_meter_change")
	if result != "5" {
		t.Errorf("Expected '5', got '%s'", result)
	}

	// 测试负数
	xmlStr = `<romance_meter_change>-3</romance_meter_change>`
	result = XmlAttr(xmlStr, "romance_meter_change")
	if result != "-3" {
		t.Errorf("Expected '-3', got '%s'", result)
	}

	// 测试零值
	xmlStr = `<romance_meter_change>0</romance_meter_change>`
	result = XmlAttr(xmlStr, "romance_meter_change")
	if result != "0" {
		t.Errorf("Expected '0', got '%s'", result)
	}
}
