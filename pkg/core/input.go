package core

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// 从父节点输出数据中提取指定路径的数据
func extractDataByPath(data any, path string) (any, error) {
	if path == "" {
		return data, nil
	}

	// 分割路径
	parts := strings.Split(path, ".")
	current := data

	// 遍历路径的每个部分
	for _, part := range parts {
		// 处理 map 类型
		if mapData, ok := current.(map[string]any); ok {
			if value, exists := mapData[part]; exists {
				current = value
				continue
			}
			return nil, fmt.Errorf("路径 %s 中的字段 %s 不存在", path, part)
		}

		// 处理数组类型
		if arrayData, ok := current.([]any); ok {
			// 尝试将部分转换为索引
			index, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("路径 %s 中的 %s 不是有效的数组索引", path, part)
			}
			if index < 0 || index >= len(arrayData) {
				return nil, fmt.Errorf("数组索引 %d 超出范围", index)
			}
			current = arrayData[index]
			continue
		}

		return nil, fmt.Errorf("路径 %s 中的 %s 不支持的数据类型: %v", path, part, reflect.TypeOf(current))
	}

	return current, nil
}

// 类型检查和转换
func validateAndConvertType(value any, expectedType []string) (any, error) {
	switch expectedType[0] {
	case "string":
		if str, ok := value.(string); ok {
			return str, nil
		}
		return nil, fmt.Errorf("类型不匹配: 期望 string, 实际是 %T", value)

	case "integer":
		// 统一解析成 int64
		if num, ok := value.(float64); ok {
			return int64(num), nil
		}
		if num, ok := value.(int64); ok {
			return int64(num), nil
		}
		if num, ok := value.(int32); ok {
			return int64(num), nil
		}
		if num, ok := value.(int); ok {
			return int64(num), nil
		}
		return nil, fmt.Errorf("类型不匹配: 期望 integer, 实际是 %T", value)

	case "float":
		if num, ok := value.(float64); ok {
			return num, nil
		}
		// 处理整数转浮点数的情况
		if num, ok := value.(int); ok {
			return float64(num), nil
		}
		return nil, fmt.Errorf("类型不匹配: 期望 float, 实际是 %T", value)

	case "boolean":
		if b, ok := value.(bool); ok {
			return b, nil
		}
		return nil, fmt.Errorf("类型不匹配: 期望 boolean, 实际是 %T", value)

	case "array":
		if arr, ok := value.([]any); ok {
			// 解析expectedType[1]元素类型
			elementType := expectedType[1]
			for _, item := range arr {
				_, err := validateAndConvertType(item, []string{elementType})
				if err != nil {
					return nil, fmt.Errorf("数组元素类型不匹配: %v", err)
				}
			}
			return arr, nil
		}
		return nil, fmt.Errorf("类型不匹配: 期望 array, 实际是 %T", value)

	case "object":
		if obj, ok := value.(map[string]any); ok {
			return obj, nil
		}
		return nil, fmt.Errorf("类型不匹配: 期望 object, 实际是 %T", value)

	default:
		return value, nil
	}
}

// ParseNodeInputs 解析节点输入
func ParseNodeInputs(inputs []Inputs, parentOutputs *ExecutionContext) (map[string]any, error) {
	result := make(map[string]any)

	for _, input := range inputs {
		// fmt.Printf("解析输入 %s 类型: %s\n,value类型: %s\n", input.Name, input.Type, input.Value.Type)
		// 如果类型是fix，则直接写入值
		if input.Value.Type == "fix" {
			result[input.Name] = input.Value.Content.Value
			// fmt.Printf("固定输入 %s 已成功解析，类型: %s\n", input.Name, input.Type)
			continue
		}
		// 获取父节点ID和输出字段名
		parentID := input.Value.Content.BlockID
		outputName := input.Value.Content.Name

		// 获取父节点的输出数据
		parentOutput, exists := parentOutputs.GetVariable(parentID + ".output")
		if !exists {
			return nil, fmt.Errorf("找不到父节点 %s 的输出数据", parentID)
		}
		// 从父节点输出中提取数据
		value, err := extractDataByPath(parentOutput, outputName)
		if err != nil {
			return nil, fmt.Errorf("解析输入 %s 时出错: %v", input.Name, err)
		}

		// 验证和转换类型
		convertedValue, err := validateAndConvertType(value, input.Type)
		if err != nil {
			return nil, fmt.Errorf("输入 %s 的类型验证失败: %v", input.Name, err)
		}

		// 将解析后的值添加到结果中
		result[input.Name] = convertedValue
		// fmt.Printf("输入 %s 已成功解析，类型: %s,值: %+v\n", input.Name, input.Type, convertedValue)
	}

	return result, nil
}
