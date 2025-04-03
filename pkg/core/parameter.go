package core

// Result 组件执行结果
type Result struct {
	Output any
	Route  []string
}

// ValidationError 验证错误
type ValidationError struct {
	Field      string `json:"field"`
	Message    string `json:"message"`
	Code       string `json:"code"`
	Severity   string `json:"severity"`
	Suggestion string `json:"suggestion"`
}

type Inputs struct {
	Name  string   `json:"name"`  // 输入的 name 定义
	Type  []string `json:"type"`  // 输入的类型
	Value Value    `json:"value"` // 输入的值
}

type Value struct {
	Content Content `json:"content"` // 输入的 content 定义
	Type    string  `json:"type"`    // 输入的类型 ref 引用  fix 固定值
}

// Content 表示值的内容
type Content struct {
	BlockID string `json:"blockId"` // 区块 ID
	Name    string `json:"name"`    // 名称
	Value   string `json:"value"`   // 值
}

// Output 表示每个输出的结构
type Output struct {
	Name      string   `json:"name"`      // 输出的名称
	Type      []string `json:"type"`      // 输出的类型
	DeftValue any      `json:"deftValue"` // 输出的默认值
	Desc      string   `json:"desc"`      // 输出的描述
	Schema    []Output `json:"schema"`    // 输出的 schema 定义
	Required  bool     `json:"required"`  // 输出的 required 定义
}
