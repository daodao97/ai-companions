package xtools

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

// XmlAttr 从XML字符串中提取指定字段的值
// 支持点分隔格式，如: "a.b.c" 表示提取 <a><b><c>value</c></b></a> 中的value
func XmlAttr(xmlStr, path string) (*gjson.Result, error) {
	// Step 1: XML => Map
	xmlMap, err := parseXMLToMap(xmlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	// Step 2: Map => JSON string
	jsonBytes, err := json.Marshal(xmlMap)
	if err != nil {
		return nil, fmt.Errorf("failed to convert map to JSON: %w", err)
	}

	// Step 3: GJson get
	result := gjson.Get(string(jsonBytes), path)
	if !result.Exists() {
		// 尝试数组的第一个元素：将路径中最后一个段前面添加.0
		parts := strings.Split(path, ".")
		if len(parts) > 1 {
			// 在倒数第二个和最后一个之间插入"0"
			newParts := make([]string, len(parts)+1)
			copy(newParts, parts[:len(parts)-1])
			newParts[len(parts)-1] = "0"
			newParts[len(parts)] = parts[len(parts)-1]
			newPath := strings.Join(newParts, ".")

			result = gjson.Get(string(jsonBytes), newPath)
			if result.Exists() {
				return &result, nil
			}
		}

		return nil, fmt.Errorf("path '%s' not found in XML", path)
	}

	return &result, nil
}

// parseXMLToMap 将XML字符串解析为map[string]interface{}
func parseXMLToMap(xmlStr string) (map[string]interface{}, error) {
	xmlStr = strings.TrimSpace(xmlStr)
	if xmlStr == "" {
		return make(map[string]interface{}), nil
	}

	decoder := xml.NewDecoder(strings.NewReader(xmlStr))
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}

	if start, ok := token.(xml.StartElement); ok {
		result := make(map[string]interface{})
		value, err := parseNode(decoder, start)
		if err != nil {
			return nil, err
		}
		result[start.Name.Local] = value
		return result, nil
	}

	return nil, fmt.Errorf("invalid XML")
}

// parseNode 解析XML节点
func parseNode(decoder *xml.Decoder, start xml.StartElement) (interface{}, error) {
	result := make(map[string]interface{})
	var text strings.Builder

	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			// 递归解析子节点
			value, err := parseNode(decoder, t)
			if err != nil {
				return nil, err
			}

			// 处理标签名映射
			name := t.Name.Local
			if name == "n" {
				name = "name"
			}

			// 处理重复标签（数组）
			if existing, exists := result[name]; exists {
				if arr, ok := existing.([]interface{}); ok {
					result[name] = append(arr, value)
				} else {
					result[name] = []interface{}{existing, value}
				}
			} else {
				result[name] = value
			}

		case xml.CharData:
			text.Write(t)

		case xml.EndElement:
			if t.Name.Local == start.Name.Local {
				if len(result) > 0 {
					return result, nil
				}
				content := strings.TrimSpace(text.String())
				return content, nil
			}
		}
	}
}
