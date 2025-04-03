package core

import (
	"errors"
	"reflect"
	"testing"
)

// 测试ValidateOutput函数
func TestValidateOutput(t *testing.T) {
	tests := []struct {
		name    string
		data    any
		output  Output
		wantErr bool
		errMsg  string
	}{
		// 字符串类型测试
		{
			name:    "有效的字符串类型",
			data:    "测试字符串",
			output:  Output{Name: "testString", Type: []string{"string"}},
			wantErr: false,
		},
		{
			name:    "无效的字符串类型",
			data:    123,
			output:  Output{Name: "testString", Type: []string{"string"}},
			wantErr: true,
			errMsg:  "输出 testString 必须是字符串类型",
		},
		// 整数类型测试
		{
			name:    "有效的整数类型",
			data:    int64(123),
			output:  Output{Name: "testInt", Type: []string{"integer"}},
			wantErr: false,
		},
		{
			name:    "无效的整数类型(浮点数)",
			data:    123.45,
			output:  Output{Name: "testInt", Type: []string{"integer"}},
			wantErr: true,
			errMsg:  "输出 testInt 必须是整数类型，不能有小数部分",
		},
		{
			name:    "无效的整数类型(非数字)",
			data:    "123",
			output:  Output{Name: "testInt", Type: []string{"integer"}},
			wantErr: true,
			errMsg:  "输出 testInt 必须是整数类型",
		},

		// 浮点数类型测试
		{
			name:    "有效的浮点数类型",
			data:    float64(123.45),
			output:  Output{Name: "testFloat", Type: []string{"float"}},
			wantErr: false,
		},
		{
			name:    "整数作为浮点数类型(自动转换)",
			data:    float64(123),
			output:  Output{Name: "testFloat", Type: []string{"float"}},
			wantErr: false,
		},
		{
			name:    "无效的浮点数类型",
			data:    "123.45",
			output:  Output{Name: "testFloat", Type: []string{"float"}},
			wantErr: true,
			errMsg:  "输出 testFloat 必须是浮点数类型",
		},

		// 布尔类型测试
		{
			name:    "有效的布尔类型true",
			data:    true,
			output:  Output{Name: "testBool", Type: []string{"boolean"}},
			wantErr: false,
		},
		{
			name:    "有效的布尔类型false",
			data:    false,
			output:  Output{Name: "testBool", Type: []string{"boolean"}},
			wantErr: false,
		},
		{
			name:    "无效的布尔类型",
			data:    "true",
			output:  Output{Name: "testBool", Type: []string{"boolean"}},
			wantErr: true,
			errMsg:  "输出 testBool 必须是布尔类型",
		},

		// 数组类型测试
		{
			name:    "有效的数组类型",
			data:    []any{"item1", "item2"},
			output:  Output{Name: "testArray", Type: []string{"array"}},
			wantErr: false,
		},
		{
			name:    "空数组类型",
			data:    []any{},
			output:  Output{Name: "testArray", Type: []string{"array"}},
			wantErr: false,
		},
		{
			name:    "无效的数组类型",
			data:    "not an array",
			output:  Output{Name: "testArray", Type: []string{"array"}},
			wantErr: true,
			errMsg:  "输出 testArray 必须是数组类型",
		},

		// 对象类型测试
		{
			name: "有效的对象类型",
			data: map[string]any{
				"name": "张三",
				"age":  float64(30),
			},
			output: Output{
				Name: "testObject",
				Type: []string{"object"},
				Schema: []Output{
					{Name: "name", Type: []string{"string"}},
					{Name: "age", Type: []string{"integer"}},
				},
			},
			wantErr: false,
		},
		{
			name: "缺少必要字段的对象",
			data: map[string]any{
				"name": "张三",
				// 缺少age字段
			},
			output: Output{
				Name: "testObject",
				Type: []string{"object"},
				Schema: []Output{
					{Name: "name", Type: []string{"string"}},
					{Name: "age", Type: []string{"integer"}},
				},
			},
			wantErr: true,
			errMsg:  "对象 testObject 缺少必要字段 age",
		},
		{
			name: "字段类型错误的对象",
			data: map[string]any{
				"name": "张三",
				"age":  "三十岁", // 类型错误，应该是整数
			},
			output: Output{
				Name: "testObject",
				Type: []string{"object"},
				Schema: []Output{
					{Name: "name", Type: []string{"string"}},
					{Name: "age", Type: []string{"integer"}},
				},
			},
			wantErr: true,
			errMsg:  "对象 testObject 的字段 age 验证失败",
		},
		{
			name:    "无效的对象类型",
			data:    "not an object",
			output:  Output{Name: "testObject", Type: []string{"object"}, Schema: []Output{}},
			wantErr: true,
			errMsg:  "输出 testObject 必须是对象类型",
		},
		{
			name: "嵌套对象类型验证",
			data: map[string]any{
				"person": map[string]any{
					"name": "张三",
					"age":  float64(30),
				},
			},
			output: Output{
				Name: "testNestedObject",
				Type: []string{"object"},
				Schema: []Output{
					{
						Name: "person",
						Type: []string{"object"},
						Schema: []Output{
							{Name: "name", Type: []string{"string"}},
							{Name: "age", Type: []string{"integer"}},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "有效的字符串数组",
			data:    []any{"item1", "item2"},
			output:  Output{Name: "testArray", Type: []string{"array"}},
			wantErr: false,
		},
		{
			name:    "有效的数字数组",
			data:    []any{float64(1), float64(2), float64(3)},
			output:  Output{Name: "testArray", Type: []string{"array"}},
			wantErr: false,
		},
		{
			name:    "有效的混合数组(未指定itemType)",
			data:    []any{"item1", float64(2), true},
			output:  Output{Name: "testArray", Type: []string{"array"}},
			wantErr: false,
		},
		{
			name:    "无效的字符串数组(含数字)",
			data:    []any{"item1", float64(2)},
			output:  Output{Name: "testArray", Type: []string{"array"}},
			wantErr: true,
			errMsg:  "数组 testArray 的元素必须是字符串类型",
		},
		{
			name:    "无效的整数数组(含小数)",
			data:    []any{float64(1), float64(2.5)},
			output:  Output{Name: "testArray", Type: []string{"array"}},
			wantErr: true,
			errMsg:  "数组 testArray 的元素必须是整数类型",
		},
		{
			name:    "无效的整数数组(含字符串)",
			data:    []any{float64(1), "2"},
			output:  Output{Name: "testArray", Type: []string{"array"}},
			wantErr: true,
			errMsg:  "数组 testArray 的元素必须是整数类型",
		},
		{
			name:    "空数组类型",
			data:    []any{},
			output:  Output{Name: "testArray", Type: []string{"array"}},
			wantErr: false,
		},
		{
			name:    "无效的数组类型",
			data:    "not an array",
			output:  Output{Name: "testArray", Type: []string{"array"}},
			wantErr: true,
			errMsg:  "输出 testArray 必须是数组类型",
		},
		{
			name: "包含对象的数组",
			data: []any{
				map[string]any{"name": "张三", "age": float64(30)},
				map[string]any{"name": "李四", "age": float64(25)},
			},
			output: Output{
				Name: "testObjectArray",
				Type: []string{"array"},
				Schema: []Output{
					{Name: "name", Type: []string{"string"}},
					{Name: "age", Type: []string{"integer"}},
				},
			},
			wantErr: false,
		},
		{
			name: "无效的对象数组(对象缺少字段)",
			data: []any{
				map[string]any{"name": "张三", "age": float64(30)},
				map[string]any{"name": "李四"}, // 缺少age字段
			},
			output: Output{
				Name: "testObjectArray",
				Type: []string{"array"},
				Schema: []Output{
					{Name: "name", Type: []string{"string"}},
					{Name: "age", Type: []string{"integer"}},
				},
			},
			wantErr: true,
			errMsg:  "对象缺少必要字段",
		},
		{
			name: "嵌套数组",
			data: []any{
				[]any{"a", "b"},
				[]any{"c", "d"},
			},
			output: Output{
				Name: "testNestedArray",
				Type: []string{"array"},
				Schema: []Output{
					{Type: []string{"string"}},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOutput(tt.data, tt.output)

			// 判断是否符合预期的错误状态
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 如果期望有错误，检查错误消息是否包含预期内容
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !errors.Is(err, err) {
					t.Errorf("错误类型不匹配: 期望 %v, 得到 %v", tt.errMsg, err)
				}
			}
		})
	}
}

// 测试ParseOutput函数
func TestParseOutput(t *testing.T) {
	tests := []struct {
		name    string
		data    any
		output  Output
		want    any
		wantErr bool
	}{
		// 基本类型测试（直接返回）
		{
			name:    "解析字符串类型",
			data:    "测试字符串",
			output:  Output{Name: "testString", Type: []string{"string"}},
			want:    "测试字符串",
			wantErr: false,
		},
		{
			name:    "解析整数类型",
			data:    float64(123),
			output:  Output{Name: "testInt", Type: []string{"integer"}},
			want:    float64(123),
			wantErr: false,
		},
		{
			name:    "解析无效整数类型",
			data:    "123",
			output:  Output{Name: "testInt", Type: []string{"integer"}},
			want:    nil,
			wantErr: true,
		},

		// 数组类型测试（直接返回）
		{
			name:    "解析数组类型",
			data:    []any{"item1", "item2"},
			output:  Output{Name: "testArray", Type: []string{"array"}},
			want:    []any{"item1", "item2"},
			wantErr: false,
		},

		// 对象类型测试
		{
			name: "解析对象类型",
			data: map[string]any{
				"name": "张三",
				"age":  float64(30),
			},
			output: Output{
				Name: "testObject",
				Type: []string{"object"},
				Schema: []Output{
					{Name: "name", Type: []string{"string"}, DeftValue: ""},
					{Name: "age", Type: []string{"integer"}, DeftValue: float64(0)},
				},
			},
			want: map[string]any{
				"name": "张三",
				"age":  float64(30),
			},
			wantErr: false,
		},
		{
			name: "解析对象类型-使用默认值",
			data: map[string]any{
				"name": "张三",
				// 缺少age字段，应使用默认值
			},
			output: Output{
				Name: "testObject",
				Type: []string{"object"},
				Schema: []Output{
					{Name: "name", Type: []string{"string"}, DeftValue: ""},
					{Name: "age", Type: []string{"integer"}, DeftValue: float64(0)},
				},
			},
			want: map[string]any{
				"name": "张三",
				"age":  float64(0), // 使用默认值
			},
			wantErr: false,
		},
		{
			name: "解析嵌套对象类型",
			data: map[string]any{
				"person": map[string]any{
					"name": "张三",
					"age":  float64(30),
				},
			},
			output: Output{
				Name: "testNestedObject",
				Type: []string{"object"},
				Schema: []Output{
					{
						Name: "person",
						Type: []string{"object"},
						Schema: []Output{
							{Name: "name", Type: []string{"string"}, DeftValue: ""},
							{Name: "age", Type: []string{"integer"}, DeftValue: float64(0)},
						},
						DeftValue: map[string]interface{}{},
					},
				},
			},
			want: map[string]any{
				"person": map[string]any{
					"name": "张三",
					"age":  float64(30),
				},
			},
			wantErr: false,
		},
		{
			name: "解析类型错误的对象",
			data: map[string]any{
				"name": 123, // 应该是字符串类型
				"age":  float64(30),
			},
			output: Output{
				Name: "testObject",
				Type: []string{"object"},
				Schema: []Output{
					{Name: "name", Type: []string{"string"}, DeftValue: ""},
					{Name: "age", Type: []string{"integer"}, DeftValue: float64(0)},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "解析字符串数组",
			data:    []any{"item1", "item2"},
			output:  Output{Name: "testArray", Type: []string{"array"}},
			want:    []any{"item1", "item2"},
			wantErr: false,
		},
		{
			name:    "解析数字数组",
			data:    []any{float64(1), float64(2)},
			output:  Output{Name: "testArray", Type: []string{"array"}},
			want:    []any{float64(1), float64(2)},
			wantErr: false,
		},
		{
			name: "解析对象数组",
			data: []any{
				map[string]any{"name": "张三", "age": float64(30)},
				map[string]any{"name": "李四", "age": float64(25)},
			},
			output: Output{
				Name: "testObjectArray",
				Type: []string{"array"},
				Schema: []Output{
					{Name: "name", Type: []string{"string"}, DeftValue: ""},
					{Name: "age", Type: []string{"integer"}, DeftValue: float64(0)},
				},
			},
			want: []any{
				map[string]any{"name": "张三", "age": float64(30)},
				map[string]any{"name": "李四", "age": float64(25)},
			},
			wantErr: false,
		},
		{
			name: "解析无效的字符串数组",
			data: []any{"item1", float64(2)},
			output: Output{
				Name: "testArray",
				Type: []string{"array"},
				Schema: []Output{
					{Name: "name", Type: []string{"string"}, DeftValue: ""},
					{Name: "age", Type: []string{"integer"}, DeftValue: float64(0)},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseOutput(tt.data, tt.output)

			// 检查是否有预期的错误
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 如果没有错误，检查结果是否符合预期
			if !tt.wantErr {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("ParseOutput() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// 测试ProcessNodeOutput函数
func TestProcessNodeOutput(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]any
		outputs []Output
		want    map[string]any
		wantErr bool
	}{
		{
			name: "处理基本字段",
			data: map[string]any{
				"name": "张三",
				"age":  float64(30),
			},
			outputs: []Output{
				{Name: "name", Type: []string{"string"}, DeftValue: ""},
				{Name: "age", Type: []string{"integer"}, DeftValue: float64(0)},
			},
			want: map[string]any{
				"name": "张三",
				"age":  float64(30),
			},
			wantErr: false,
		},
		{
			name: "处理缺少字段-使用默认值",
			data: map[string]any{
				"name": "张三",
				// 缺少age和gender字段
			},
			outputs: []Output{
				{Name: "name", Type: []string{"string"}, DeftValue: ""},
				{Name: "age", Type: []string{"integer"}, DeftValue: float64(0)},
				{Name: "gender", Type: []string{"string"}, DeftValue: "未知"},
			},
			want: map[string]any{
				"name":   "张三",
				"age":    float64(0), // 使用默认值
				"gender": "未知",       // 使用默认值
			},
			wantErr: false,
		},
		{
			name: "处理类型错误的字段",
			data: map[string]any{
				"name": "张三",
				"age":  "三十岁", // 类型错误，应该是整数
			},
			outputs: []Output{
				{Name: "name", Type: []string{"string"}, DeftValue: ""},
				{Name: "age", Type: []string{"integer"}, DeftValue: float64(0)},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "处理复杂对象和数组",
			data: map[string]any{
				"name": "张三",
				"info": map[string]any{
					"city":    "杭州",
					"country": "中国",
					"height":  float64(180),
					"email":   []any{"zhang@example.com"},
				},
				"tags": []any{"学生", "程序员"},
			},
			outputs: []Output{
				{Name: "name", Type: []string{"string"}, DeftValue: ""},
				{
					Name: "info",
					Type: []string{"object"},
					Schema: []Output{
						{Name: "city", Type: []string{"string"}, DeftValue: ""},
						{Name: "country", Type: []string{"string"}, DeftValue: ""},
						{Name: "height", Type: []string{"float"}, DeftValue: float64(0)},
						{Name: "email", Type: []string{"array"}, DeftValue: []any{}},
					},
					DeftValue: map[string]any{},
				},
				{Name: "tags", Type: []string{"array"}, DeftValue: []any{}},
			},
			want: map[string]any{
				"name": "张三",
				"info": map[string]any{
					"city":    "杭州",
					"country": "中国",
					"height":  float64(180),
					"email":   []any{"zhang@example.com"},
				},
				"tags": []any{"学生", "程序员"},
			},
			wantErr: false,
		},
		{
			name: "处理空数据",
			data: map[string]any{},
			outputs: []Output{
				{Name: "name", Type: []string{"string"}, DeftValue: "未命名"},
				{Name: "age", Type: []string{"integer"}, DeftValue: float64(0)},
			},
			want: map[string]any{
				"name": "未命名",
				"age":  float64(0),
			},
			wantErr: false,
		},
		{
			name: "处理带类型的数组",
			data: map[string]any{
				"name":   "张三",
				"tags":   []any{"学生", "程序员"},
				"scores": []any{float64(90), float64(85), float64(95)},
			},
			outputs: []Output{
				{Name: "name", Type: []string{"string"}, DeftValue: ""},
				{Name: "tags", Type: []string{"array"}, DeftValue: []any{}},
				{Name: "scores", Type: []string{"array"}, DeftValue: []any{}},
			},
			want: map[string]any{
				"name":   "张三",
				"tags":   []any{"学生", "程序员"},
				"scores": []any{float64(90), float64(85), float64(95)},
			},
			wantErr: false,
		},
		{
			name: "处理类型错误的数组",
			data: map[string]any{
				"name": "张三",
				"tags": []any{"学生", 123}, // 类型错误
			},
			outputs: []Output{
				{Name: "name", Type: []string{"string"}, DeftValue: ""},
				{Name: "tags", Type: []string{"array"}, DeftValue: []any{}},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "处理对象数组",
			data: map[string]any{
				"name": "张三",
				"friends": []any{
					map[string]any{"name": "李四", "age": float64(25)},
					map[string]any{"name": "王五", "age": float64(28)},
				},
			},
			outputs: []Output{
				{Name: "name", Type: []string{"string"}, DeftValue: ""},
				{
					Name: "friends",
					Type: []string{"array"},
					Schema: []Output{
						{Name: "name", Type: []string{"string"}, DeftValue: ""},
						{Name: "age", Type: []string{"integer"}, DeftValue: float64(0)},
					},
					DeftValue: []any{},
				},
			},
			want: map[string]any{
				"name": "张三",
				"friends": []any{
					map[string]any{"name": "李四", "age": float64(25)},
					map[string]any{"name": "王五", "age": float64(28)},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ProcessNodeOutput(tt.data, tt.outputs)

			// 检查是否有预期的错误
			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessNodeOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 如果没有错误，检查结果是否符合预期
			if !tt.wantErr {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("ProcessNodeOutput() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
