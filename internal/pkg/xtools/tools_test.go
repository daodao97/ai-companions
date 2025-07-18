package xtools

import (
	"testing"
)

func TestXmlAttr(t *testing.T) {
	tests := []struct {
		name     string
		xmlStr   string
		path     string
		expected string
		wantErr  bool
	}{
		{
			name:     "简单标签提取",
			xmlStr:   `<root><name>张三</name><age>25</age></root>`,
			path:     "root.name",
			expected: "张三",
			wantErr:  false,
		},
		{
			name:     "嵌套标签提取",
			xmlStr:   `<root><user><profile><name>李四</name><age>30</age></profile></user></root>`,
			path:     "root.user.profile.name",
			expected: "李四",
			wantErr:  false,
		},
		{
			name:     "数组元素提取",
			xmlStr:   `<root><users><user><name>王五</name></user><user><name>赵六</name></user></users></root>`,
			path:     "root.users.user.name",
			expected: "王五",
			wantErr:  false,
		},
		{
			name:     "空路径",
			xmlStr:   `<root><name>测试</name></root>`,
			path:     "",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "空XML",
			xmlStr:   "",
			path:     "root.name",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "不存在的路径",
			xmlStr:   `<root><name>测试</name></root>`,
			path:     "root.age",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "无效XML",
			xmlStr:   `<root><name>测试</name>`,
			path:     "root.name",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := XmlAttr(tt.xmlStr, tt.path)

			if tt.wantErr {
				if err == nil {
					t.Errorf("XmlAttr() 期望错误但没有返回错误")
				}
				return
			}

			if err != nil {
				t.Errorf("XmlAttr() 返回了意外的错误: %v", err)
				return
			}

			if result.String() != tt.expected {
				t.Errorf("XmlAttr() = %v, 期望 %v", result.String(), tt.expected)
			}
		})
	}
}

func TestXmlAttrComplex(t *testing.T) {
	complexXML := `
	<response>
		<status>success</status>
		<data>
			<users>
				<user>
					<id>1</id>
					<name>张三</name>
					<email>zhangsan@example.com</email>
				</user>
				<user>
					<id>2</id>
					<name>李四</name>
					<email>lisi@example.com</email>
				</user>
			</users>
			<total>2</total>
		</data>
	</response>
	`

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"状态提取", "response.status", "success"},
		{"用户数量", "response.data.total", "2"},
		{"第一个用户名", "response.data.users.user.name", "张三"},
		{"第一个用户邮箱", "response.data.users.user.email", "zhangsan@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := XmlAttr(complexXML, tt.path)
			if err != nil {
				t.Errorf("XmlAttr() 返回了意外的错误: %v", err)
				return
			}

			if result.String() != tt.expected {
				t.Errorf("XmlAttr() = %v, 期望 %v", result.String(), tt.expected)
			}
		})
	}
}
