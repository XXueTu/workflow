package core

import (
	"time"
)

// WorkflowDef 工作流定义
type WorkflowDef struct {
	// 基础信息
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	CreateTime  time.Time `json:"createTime"`
	UpdateTime  time.Time `json:"updateTime"`

	// 节点定义
	Nodes       map[string]*NodeDefinition `json:"nodes"`
	Connections []Connection               `json:"connections"`

	// 配置信息
	Config WorkflowConfig `json:"config"`
}

// Connection 节点连接
type Connection struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Condition string `json:"condition,omitempty"`
}

// WorkflowConfig 工作流配置
type WorkflowConfig struct {
	Timeout     time.Duration `json:"timeout"`
	RetryPolicy *RetryPolicy  `json:"retryPolicy,omitempty"`
}

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxAttempts int           `json:"maxAttempts"`
	Interval    time.Duration `json:"interval"`
	MaxInterval time.Duration `json:"maxInterval"`
	Multiplier  float64       `json:"multiplier"`
}

// NodeDefinition 节点定义
type NodeDefinition struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Inputs  []Inputs `json:"inputs"`
	Outputs []Output `json:"outputs"`
	// 组件配置
	Config map[string]any `json:"config"`
	// 迭代
	SubWorkflow WorkflowDef `json:"subWorkflow"`
}
