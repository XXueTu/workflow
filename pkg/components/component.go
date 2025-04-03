package components

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/XXueTu/workflow/pkg/core"
)

const (
	Success = "success"
	Failed  = "failed"

	Skip = "skip"

	True  = "true"
	False = "false"
)

const (
	Start     = "start"
	End       = "end"
	HTTP      = "http"
	Codejs    = "codejs"
	Branch    = "branch"
	Iteration = "iteration"
	StartItem = "start-item"
	EndItem   = "end-item"
)

// Component 定义组件核心接口
type Component interface {
	Validate() []core.ValidationError
	AnalyzeInputs(ctx context.Context) (any, error)
	Execute(ctx context.Context, input any) (*core.Result, error)
}

// ComponentFactory 组件工厂
func ComponentFactory(e core.WorkflowEngine, nodeType string, nodeConfig *core.NodeDefinition) (Component, error) {
	jsonConfig, err := json.Marshal(nodeConfig.Config)
	if err != nil {
		return nil, fmt.Errorf("component configuration serialization failed: %v", err)
	}
	switch nodeType {
	case Start:
		return NewStartComponent()
	case End:
		return NewEndComponent(jsonConfig)
	case HTTP:
		return NewHTTPComponent(jsonConfig)
	case Codejs:
		return NewCodejsComponent(jsonConfig)
	case Branch:
		return NewBranchComponent(jsonConfig)
	case Iteration:
		return NewIterationComponent(e, jsonConfig)
	case StartItem:
		return NewStartItemComponent()
	case EndItem:
		return NewEndItemComponent(jsonConfig)
	}
	return nil, fmt.Errorf("Component type not found: %s", nodeType)
}
