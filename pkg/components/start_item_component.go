package components

import (
	"context"
	"sync"

	"github.com/XXueTu/workflow/pkg/core"
)

// StartItemComponent 迭代启动组件
type StartItemComponent struct {
}

var startItemComponentPool = sync.Pool{
	New: func() interface{} {
		return &StartItemComponent{}
	},
}

func NewStartItemComponent() (*StartItemComponent, error) {
	c := startItemComponentPool.Get().(*StartItemComponent)
	return c, nil
}

func (c *StartItemComponent) Execute(ctx context.Context, input any) (*core.Result, error) {
	return &core.Result{
		Route:  []string{Success},
		Output: input,
	}, nil
}

func (c *StartItemComponent) Validate() []core.ValidationError {
	return nil
}

func (c *StartItemComponent) AnalyzeInputs(ctx context.Context) (any, error) {
	return nil, nil
}
