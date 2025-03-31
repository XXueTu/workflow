package components

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/XXueTu/workflow/internal/core"
)

// EndComponent 结束组件
type EndComponent struct {
	config EndConfig
}

type EndConfig struct {
	OutPutType string `json:"outPutType"` // 输出类型 json
}

var endComponentPool = sync.Pool{
	New: func() interface{} {
		return &EndComponent{}
	},
}

func NewEndComponent(config json.RawMessage) (*EndComponent, error) {
	var endConfig EndConfig
	if err := json.Unmarshal(config, &endConfig); err != nil {
		return nil, errors.New("解析结束组件配置失败: " + err.Error())
	}
	return &EndComponent{config: endConfig}, nil
}

func (c *EndComponent) Execute(ctx context.Context, input any) (*core.Result, error) {
	fmt.Printf("结束组件执行: %+v\n", input)
	return &core.Result{
		Route:  []string{Success},
		Output: input,
	}, nil
}

func (c *EndComponent) Validate() []core.ValidationError {
	return nil
}

func (c *EndComponent) AnalyzeInputs(ctx context.Context) (any, error) {
	return nil, nil
}
