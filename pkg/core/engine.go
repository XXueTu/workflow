package core

import (
	"context"
	"time"
)

type WorkflowEngine interface {
	ExecuteWorkflow(ctx context.Context, workflowID string, serialID string, params map[string]any) error
	GetNodeResult(workflowID, serialID, nodeID string) (*NodeResult, bool)

	ListWorkflows() []string
	GetWorkflowStatus(workflowID string) (WorkflowStatus, bool)
	GetPoolStats() map[string]any
	GetMetrics(workflowID string) (*WorkflowMetrics, bool)

	PauseWorkflow(ctx context.Context, workflowID string, serialID string) error
}

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
	PausedExecutions     int64
	AverageExecutionTime float64
	LastExecutionTime    time.Time
}
