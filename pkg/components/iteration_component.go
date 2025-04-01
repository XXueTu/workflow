package components

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/XXueTu/workflow/pkg/core"
)

type IterationComponent struct {
	config IterationConfig
}

type IterationConfig struct {
	IterationType  string // array、index 0-
	IterationValue core.Inputs
	SubWorkflowId  string
	workflowEngine core.WorkflowEngine
}

var iterationComponentPool = sync.Pool{
	New: func() interface{} {
		return &IterationComponent{}
	},
}

func NewIterationComponent(e core.WorkflowEngine, config json.RawMessage) (*IterationComponent, error) {
	c := iterationComponentPool.Get().(*IterationComponent)
	var iteraConfig IterationConfig
	if err := json.Unmarshal(config, &iteraConfig); err != nil {
		return nil, fmt.Errorf("解析迭代执行组件配置失败: %v", err)
	}
	iteraConfig.workflowEngine = e
	c.config = iteraConfig
	return c, nil
}

const (
	iterVal = "iterVal"
)

// AnalyzeInputs implements Component.
func (i *IterationComponent) AnalyzeInputs(ctx context.Context) (any, error) {
	execCtx := ctx.(*core.ExecutionContext)
	valMap, err := core.ParseNodeInputs([]core.Inputs{i.config.IterationValue}, execCtx)
	if err != nil {
		return nil, fmt.Errorf("解析迭代值失败: %w", err)
	}

	if len(valMap) == 0 {
		return nil, fmt.Errorf("迭代值为空")
	}

	// 获取第一个值
	var value any
	for _, v := range valMap {
		value = v
		break
	}

	input := map[string]any{}
	switch i.config.IterationType {
	case "array":
		arr, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("迭代值必须是数组类型")
		}
		input[iterVal] = arr
	case "index":
		num, ok := value.(int64)
		if !ok {
			return nil, fmt.Errorf("迭代值必须是数字类型")
		}
		input[iterVal] = num
	default:
		return nil, fmt.Errorf("不支持的迭代类型: %s", i.config.IterationType)
	}

	return input, nil
}

// Execute implements Component.
func (i *IterationComponent) Execute(ctx context.Context, input any) (*core.Result, error) {
	switch i.config.IterationValue.Type {
	case "array":
		{
			// 封装参数
			inputMap, ok := input.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("输入类型不匹配")
			}
			iterVal, ok := inputMap[iterVal].([]any)
			if !ok {
				return nil, fmt.Errorf("迭代值类型不匹配")
			}
			for i, v := range iterVal {
				fmt.Printf("迭代内容: %d,%+v\n", i, v)
			}
		}
	}
	result := core.Result{
		Output: []string{"a", "b", "c"},
		Route:  []string{Success},
	}
	return &result, nil
}

// Validate implements Component.
func (i *IterationComponent) Validate() []core.ValidationError {
	return nil
}

// var _ Component = new(IterationComponent)
