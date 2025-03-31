package components

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/XXueTu/workflow/internal/core"
)

// StartComponent 启动组件
type BranchComponent struct {
	config BranchConfig
}

type BranchConfig struct {
	Conditions []Condition `json:"conditions"`
}

type Condition struct {
	Left     core.Inputs `json:"left"`
	Right    core.Inputs `json:"right"`
	Route    string      `json:"route"`
	Operator string      `json:"operator"`
}

const (
	OperatorEquals    = "eq"
	OperatorNotEquals = "ne"
	OperatorIn        = "in"
	OperatorNotIn     = "nin"
	OperatorIsNull    = "isNull"
	OperatorIsNotNull = "isNotNull"
	OperatorGt        = "gt"
	OperatorLt        = "lt"
	OperatorGe        = "ge"
	OperatorLe        = "le"
)

type Cp struct {
	Left     any    `json:"left"`
	Right    any    `json:"right"`
	Route    string `json:"route"`
	Operator string `json:"operator"`
}

var branchComponentPool = sync.Pool{
	New: func() interface{} {
		return &BranchComponent{}
	},
}

func NewBranchComponent(config json.RawMessage) (*BranchComponent, error) {
	var branchConfig BranchConfig
	if err := json.Unmarshal(config, &branchConfig); err != nil {
		return nil, errors.New("解析分支组件配置失败: " + err.Error())
	}
	c := branchComponentPool.Get().(*BranchComponent)
	c.config = branchConfig

	return c, nil
}

func (c *BranchComponent) Execute(ctx context.Context, input any) (*core.Result, error) {
	inputArray, ok := input.([]Cp)
	output := make(map[string]any)
	output["branch"] = inputArray
	if !ok {
		return nil, errors.New("输入类型不匹配")
	}
	var route = []string{}
	for _, condition := range inputArray {
		fmt.Printf("branch execute condition: %+v\n", condition)
		switch condition.Operator {
		case OperatorEquals:
			if condition.Left == condition.Right {
				route = append(route, condition.Route)
			}
		case OperatorNotEquals:
			if condition.Left != condition.Right {
				route = append(route, condition.Route)
			}
		case OperatorGt:
			// 比较数值
			left, right, err := compareValues(condition.Left, condition.Right)
			if err != nil {
				fmt.Println("比较数值失败")
				continue
			}
			if left <= right {
				fmt.Println("左值不大于右值")
				continue
			}
			route = append(route, condition.Route)
		case OperatorLt:
			// 比较数值
			left, right, err := compareValues(condition.Left, condition.Right)
			if err != nil {
				fmt.Println("比较数值失败")
				continue
			}
			if left >= right {
				fmt.Println("左值不小于右值")
				continue
			}
			route = append(route, condition.Route)
		case OperatorGe:
			// 比较数值
			left, right, err := compareValues(condition.Left, condition.Right)
			if err != nil {
				fmt.Println("比较数值失败")
				continue
			}
			if left < right {
				fmt.Println("左值小于右值")
				continue
			}
			route = append(route, condition.Route)
		case OperatorLe:
			// 比较数值
			left, right, err := compareValues(condition.Left, condition.Right)
			if err != nil {
				return nil, err
			}
			if left > right {
				fmt.Println("左值大于右值")
				continue
			}
			route = append(route, condition.Route)
		case OperatorIn:
			// 检查左值是否在右值数组中
			rightArray, ok := condition.Right.([]any)
			if !ok {
				return &core.Result{
					Route:  []string{False},
					Output: output,
				}, errors.New("IN 右值类型不匹配,不是数组")
			}

			// 如果左值是数组，检查是否有任意一个元素在右值数组中
			if leftArray, ok := condition.Left.([]any); ok {
				found := false
				for _, left := range leftArray {
					if slices.Contains(rightArray, left) {
						found = true
						break
					}
				}
				if !found {
					fmt.Println("左值数组中没有元素在右值数组中")
					continue
				}
			} else {
				// 如果左值是单个值，直接检查是否在右值数组中
				if !slices.Contains(rightArray, condition.Left) {
					fmt.Println("左值不在右值数组中")
					continue
				}
			}
			route = append(route, condition.Route)
		case OperatorNotIn:
			// 检查左值是否不在右值数组中
			rightArray, ok := condition.Right.([]any)
			if !ok {
				return &core.Result{
					Route:  []string{False},
					Output: output,
				}, errors.New("NOT IN 右值类型不匹配,不是数组")
			}

			// 如果左值是数组，检查是否所有元素都不在右值数组中
			if leftArray, ok := condition.Left.([]any); ok {
				for _, left := range leftArray {
					if slices.Contains(rightArray, left) {
						fmt.Println("左值数组中有元素在右值数组中")
						continue
					}
				}
			} else {
				// 如果左值是单个值，直接检查是否不在右值数组中
				if slices.Contains(rightArray, condition.Left) {
					fmt.Println("左值在右值数组中")
					continue
				}
			}
			route = append(route, condition.Route)
		case OperatorIsNull:
			if condition.Left != nil {
				fmt.Println("左值不为空")
				continue
			}
			route = append(route, condition.Route)
		case OperatorIsNotNull:
			if condition.Left == nil {
				fmt.Println("左值为空")
				continue
			}
			route = append(route, condition.Route)
		}
	}
	if len(route) == 0 {
		fmt.Println("没有满足条件的路由")
		route = append(route, False)
	}
	return &core.Result{
		Route:  route,
		Output: output,
	}, nil
}

func (c *BranchComponent) Validate() []core.ValidationError {
	if len(c.config.Conditions) == 0 {
		return []core.ValidationError{
			{
				Field:   "conditions",
				Message: "条件不能为空",
			},
		}
	}
	for _, condition := range c.config.Conditions {
		if condition.Route == "" {
			return []core.ValidationError{
				{
					Field:   "route",
					Message: "路由不能为空",
				},
			}
		}
		if condition.Operator == "" {
			return []core.ValidationError{
				{
					Field:   "operator",
					Message: "操作符不能为空",
				},
			}
		}
	}
	return nil
}
func (c *BranchComponent) AnalyzeInputs(ctx context.Context) (any, error) {
	execCtx := ctx.(*core.ExecutionContext)
	conditions := make([]Cp, len(c.config.Conditions))
	for i, condition := range c.config.Conditions {
		left, err := core.ParseNodeInputs([]core.Inputs{condition.Left}, execCtx)
		if err != nil {
			return nil, err
		}
		right, err := core.ParseNodeInputs([]core.Inputs{condition.Right}, execCtx)
		if err != nil {
			return nil, err
		}
		conditions[i] = Cp{
			Left:     left[condition.Left.Name],
			Right:    right[condition.Right.Name],
			Route:    condition.Route,
			Operator: condition.Operator,
		}
	}
	fmt.Printf("conditions: %+v\n", conditions)
	return conditions, nil
}

// compareValues 比较两个值
func compareValues(left, right any) (float64, float64, error) {
	// 转换为float64进行比较
	var leftFloat, rightFloat float64

	switch v := left.(type) {
	case int:
		leftFloat = float64(v)
	case int64:
		leftFloat = float64(v)
	case float32:
		leftFloat = float64(v)
	case float64:
		leftFloat = v
	default:
		return 0, 0, fmt.Errorf("不支持的左值类型: %T", left)
	}

	switch v := right.(type) {
	case int:
		rightFloat = float64(v)
	case int64:
		rightFloat = float64(v)
	case float32:
		rightFloat = float64(v)
	case float64:
		rightFloat = v
	default:
		return 0, 0, fmt.Errorf("不支持的右值类型: %T", right)
	}

	return leftFloat, rightFloat, nil
}
