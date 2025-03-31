package components

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/XXueTu/workflow/flow-engine/internal/core"
)

const (
	Success = "success"
	Failed  = "failed"

	Skip = "skip"

	True  = "true"
	False = "false"
)

// Component 定义组件核心接口
type Component interface {
	Validate() []core.ValidationError
	AnalyzeInputs(ctx context.Context) (any, error)
	Execute(ctx context.Context, input any) (*core.Result, error)
}

// ComponentFactory 组件工厂
func ComponentFactory(nodeType string, nodeConfig *core.NodeDefinition) (Component, error) {
	jsonConfig, err := json.Marshal(nodeConfig.Config)
	if err != nil {
		return nil, fmt.Errorf("component configuration serialization failed: %v", err)
	}
	switch nodeType {
	case "start":
		return NewStartComponent()
	case "end":
		return NewEndComponent(jsonConfig)
	case "http":
		return NewHTTPComponent(jsonConfig)
	case "codejs":
		return NewCodejsComponent(jsonConfig)
	case "branch":
		return NewBranchComponent(jsonConfig)
	}
	return nil, fmt.Errorf("Component type not found: %s", nodeType)
}
