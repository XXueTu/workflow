package engine

import (
	"sync"
	"time"

	"github.com/XXueTu/workflow/pkg/components"
	"github.com/XXueTu/workflow/pkg/core"
)

// WorkflowStatus 表示工作流的状态
type WorkflowStatus string

const (
	WorkflowStatusActive    WorkflowStatus = "active"
	WorkflowStatusDeploying WorkflowStatus = "deploying"
	WorkflowStatusShutdown  WorkflowStatus = "shutdown"
)

// WorkflowMetrics 工作流指标
type WorkflowMetrics struct {
	ActiveExecutions     int64
	CompletedExecutions  int64
	FailedExecutions     int64
	AverageExecutionTime float64
	LastExecutionTime    time.Time
}

type EngineConfig struct {
	InitialPoolSize        int           // 初始池大小
	MaxPoolSize            int           // 最大池大小
	ExecutionTimeout       time.Duration // 执行超时
	CleanupInterval        time.Duration // 清理间隔
	DefaultContextTTL      time.Duration // 上下文TTL
	EnableMetrics          bool          // 启用指标
	EnableTracing          bool          // 启用跟踪
	MaxConcurrentWorkflows int           // 最大并发工作流
}

// MetricsCollector 指标收集器
type MetricsCollector struct {
	metrics map[string]*WorkflowMetrics
	mu      sync.RWMutex
}

type Executor struct {
	definition      *core.WorkflowDef
	executionPlan   *ExecutionPlan
	conditionRouter map[string][]string
	execContexts    sync.Map // 使用sync.Map替代map+mutex提高并发性能
	totalNodes      int64
	env             map[string]any
	status          WorkflowStatus   // 工作流状态
	metrics         *WorkflowMetrics // 工作流指标
	activeCount     int64            // 活跃执行计数
	lastAccessed    int64            // 最后访问时间戳
	createdAt       time.Time        // 创建时间
	defaultTTL      time.Duration    // 默认TTL
	shutdownCh      chan struct{}    // 关闭通道
}

// ExecutionTask 表示一个执行任务
type ExecutionTask struct {
	NodeID     string
	Component  components.Component
	Context    *core.ExecutionContext
	Engine     *WorkflowEngine
	Executor   *Executor
	WorkflowID string
	SerialID   string
	Sw         *sync.WaitGroup
}

// ExecutionPlan 执行计划
type ExecutionPlan struct {
	Phases []ExecutionPhase
}

// ExecutionPhase 执行阶段
type ExecutionPhase struct {
	Nodes []*WorkflowNode
}

// WorkflowNode 工作流节点
type WorkflowNode struct {
	*core.NodeDefinition
	Component components.Component
}

// DefaultConfig 返回默认配置
func DefaultConfig() *EngineConfig {
	return &EngineConfig{
		InitialPoolSize:        10,
		MaxPoolSize:            100,
		ExecutionTimeout:       5 * time.Minute,
		CleanupInterval:        1 * time.Minute,
		DefaultContextTTL:      24 * time.Hour,
		EnableMetrics:          true,
		EnableTracing:          true,
		MaxConcurrentWorkflows: 1000,
	}
}

// Option 是工作流引擎的配置选项
type Option func(*WorkflowEngine)

// WithPoolSize 设置池大小
func WithPoolSize(size int) Option {
	return func(e *WorkflowEngine) {
		e.config.InitialPoolSize = size
	}
}

// WithMaxPoolSize 设置最大池大小
func WithMaxPoolSize(size int) Option {
	return func(e *WorkflowEngine) {
		e.config.MaxPoolSize = size
	}
}

// WithExecutionTimeout 设置执行超时
func WithExecutionTimeout(timeout time.Duration) Option {
	return func(e *WorkflowEngine) {
		e.config.ExecutionTimeout = timeout
	}
}

// WithCleanupInterval 设置清理间隔
func WithCleanupInterval(interval time.Duration) Option {
	return func(e *WorkflowEngine) {
		e.config.CleanupInterval = interval
	}
}

// WithDefaultContextTTL 设置上下文TTL
func WithDefaultContextTTL(ttl time.Duration) Option {
	return func(e *WorkflowEngine) {
		e.config.DefaultContextTTL = ttl
	}
}

// WithMetricsEnabled 启用指标
func WithMetricsEnabled(enabled bool) Option {
	return func(e *WorkflowEngine) {
		e.config.EnableMetrics = enabled
	}
}

// WithTracingEnabled 启用跟踪
func WithTracingEnabled(enabled bool) Option {
	return func(e *WorkflowEngine) {
		e.config.EnableTracing = enabled
	}
}

// WithMaxConcurrentWorkflows 设置最大并发工作流
func WithMaxConcurrentWorkflows(max int) Option {
	return func(e *WorkflowEngine) {
		e.config.MaxConcurrentWorkflows = max
	}
}
