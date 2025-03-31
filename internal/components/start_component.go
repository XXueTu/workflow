package components

import (
	"context"
	"sync"

	"github.com/XXueTu/workflow/flow-engine/internal/core"
)

// StartComponent 启动组件
type StartComponent struct {
}

var startComponentPool = sync.Pool{
	New: func() interface{} {
		return &StartComponent{}
	},
}

func NewStartComponent() (*StartComponent, error) {
	c := startComponentPool.Get().(*StartComponent)
	return c, nil
}

func (c *StartComponent) Execute(ctx context.Context, input any) (*core.Result, error) {
	return &core.Result{
		Route:  []string{Success},
		Output: input,
	}, nil
}

func (c *StartComponent) Validate() []core.ValidationError {
	return nil
}

func (c *StartComponent) AnalyzeInputs(ctx context.Context) (any, error) {
	return nil, nil
}
