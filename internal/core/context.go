package core

import (
	"context"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
)

// ExecutionContext 工作流执行上下文
type ExecutionContext struct {
	context.Context
	Id        string
	variables map[string]interface{}
	State     *WorkflowState
	Route     map[string]struct{}
	mu        sync.RWMutex
	tracer    opentracing.Tracer
	startTime time.Time
	metrics   map[string]float64

	TotalNodes int64
	Expiration time.Time // 添加过期时间
}

type WorkflowState struct {
	Status   ExecutionStatus
	Progress float64
	Result   map[string]*NodeResult
	Errors   []error
}

type NodeResult struct {
	Input    any
	Output   any
	Route    []string
	NodeID   string
	Duration int64 // ms
	Error    string
	Type     string
}

// ExecutionStatus 执行状态
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusPaused    ExecutionStatus = "paused"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusCanceled  ExecutionStatus = "canceled"
)

var execContextPool = sync.Pool{
	New: func() interface{} {
		return &ExecutionContext{}
	},
}

// NewExecutionContext 创建新的执行上下文
func NewExecutionContext(ctx context.Context, serialID string, totalNodes int64, params map[string]any) *ExecutionContext {
	execCtx := execContextPool.Get().(*ExecutionContext)
	execCtx.Context = ctx
	execCtx.Id = serialID
	execCtx.TotalNodes = totalNodes
	execCtx.State = NewWorkflowState()
	execCtx.Route = make(map[string]struct{})
	execCtx.startTime = time.Now()
	execCtx.Expiration = time.Now().Add(5 * time.Minute) // 设置过期时间
	execCtx.metrics = make(map[string]float64)
	execCtx.variables = make(map[string]any, totalNodes+1)
	execCtx.variables["_zero"] = params
	return execCtx
}

// ReleaseExecutionContext 释放执行上下文
func ReleaseExecutionContext(ctx *ExecutionContext) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	// 清理状态
	ctx.State = nil
	ctx.variables = make(map[string]any)
	execContextPool.Put(ctx) // 将上下文放回池中
}

// ContextOption 上下文配置选项
type ContextOption func(*ExecutionContext)

// WithTracer 设置追踪器
func WithTracer(tracer opentracing.Tracer) ContextOption {
	return func(ctx *ExecutionContext) {
		// No-op, as the context is now a context.Context
	}
}

// GetVariable 获取变量值
func (ctx *ExecutionContext) GetVariable(key string) (any, bool) {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	value, ok := ctx.variables[key]
	return value, ok
}

// SetVariable 设置变量值
func (ctx *ExecutionContext) SetVariable(key string, value any) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.variables[key] = value
}

// CheckRoute 检查路由
func (ctx *ExecutionContext) CheckRoute(key []string) bool {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	for _, route := range key {
		_, ok := ctx.Route[route]
		if ok {
			return true
		}
	}
	return false
}

// Cancel 取消执行
func (ctx *ExecutionContext) Cancel() {
	// No-op, as the context is now a context.Context
}

// Done 实现 context.Context 接口
func (c *ExecutionContext) Done() <-chan struct{} {
	// 直接返回底层 Context 的 Done channel
	return c.Context.Done()
}

// Deadline 实现 context.Context 接口
func (c *ExecutionContext) Deadline() (deadline time.Time, ok bool) {
	return c.Context.Deadline()
}

// Err 实现 context.Context 接口
func (c *ExecutionContext) Err() error {
	return c.Context.Err()
}

// Value 实现 context.Context 接口
func (c *ExecutionContext) Value(key interface{}) interface{} {
	if key == "execution_context" {
		return c
	}
	return c.Context.Value(key)
}

func (ctx *ExecutionContext) SetTracer(tracer opentracing.Tracer) {
	ctx.tracer = tracer
}

func (ctx *ExecutionContext) SetError(nodeID string, err error) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.State.Errors = append(ctx.State.Errors, err)
}

func (ctx *ExecutionContext) SetNodeResult(nodeID string, result *NodeResult) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.State.Result[nodeID] = result
	for _, route := range result.Route {
		ctx.Route[nodeID+"_"+route] = struct{}{}
	}
}

func NewWorkflowState() *WorkflowState {
	return &WorkflowState{
		Status: StatusPending,
		Errors: make([]error, 0),
		Result: make(map[string]*NodeResult),
	}
}

// GetNodeResult 获取指定节点的执行结果
func (ctx *ExecutionContext) GetNodeResult(nodeID string) (*NodeResult, bool) {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	if ctx.State == nil || ctx.State.Result == nil {
		return nil, false
	}
	state, exists := ctx.State.Result[nodeID]
	if !exists || state == nil {
		return nil, false
	}
	return state, true
}

// GetAllResults 获取所有节点的执行结果
func (ctx *ExecutionContext) GetAllResults() map[string]*NodeResult {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	results := make(map[string]*NodeResult)
	if ctx.State == nil || ctx.State.Result == nil {
		return results
	}

	for nodeID, state := range ctx.State.Result {
		if state != nil {
			results[nodeID] = state
		}
	}
	return results
}
