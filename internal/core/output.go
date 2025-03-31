package core

import (
	"encoding/json"
	"fmt"
)

// ValidateOutput 验证输出数据是否符合Schema定义
func ValidateOutput(data any, output Output) error {
	switch output.Type {
	case "string":
		if _, ok := data.(string); !ok {
			return fmt.Errorf("输出 %s 必须是字符串类型", output.Name)
		}
	case "integer":
		switch v := data.(type) {
		case int:
			if float64(v) != float64(v) {
				return fmt.Errorf("输出 %s int必须是整数类型，不能有小数部分", output.Name)
			}
		case int32:
			if float64(v) != float64(v) {
				return fmt.Errorf("输出 %s int32必须是整数类型，不能有小数部分", output.Name)
			}
		case int64:
			if float64(v) != float64(v) {
				return fmt.Errorf("输出 %s int64必须是整数类型，不能有小数部分", output.Name)
			}
		case float64:
			if v != float64(int(v)) {
				return fmt.Errorf("输出 %s float64必须是整数类型，不能有小数部分", output.Name)
			}
		default:
			return fmt.Errorf("输出 %s ,%T必须是整数类型", output.Name, v)
		}
	case "float":
		if _, ok := data.(float64); !ok {
			return fmt.Errorf("输出 %s 必须是浮点数类型", output.Name)
		}
	case "boolean":
		if _, ok := data.(bool); !ok {
			return fmt.Errorf("输出 %s 必须是布尔类型", output.Name)
		}
	case "array":
		arr, ok := data.([]any)
		if !ok {
			return fmt.Errorf("输出 %s 必须是数组类型", output.Name)
		}
		// 如果有更复杂的数组元素验证，可以在这里添加
		if output.ItemType != "" {
			if len(arr) > 0 {
				err := ValidateOutput(arr[0], Output{Type: output.ItemType})
				if err != nil {
					return fmt.Errorf("数组 %s 的元素 %v 验证失败: %v", output.Name, arr[0], err)
				}
			}
		}
	case "object":
		obj, ok := data.(map[string]any)
		if !ok {
			return fmt.Errorf("输出 %s 必须是对象类型,现在: %T", output.Name, data)
		}

		// 验证对象的每个字段
		for _, field := range output.Schema {
			fieldValue, exists := obj[field.Name]
			if !exists {
				return fmt.Errorf("对象 %s 缺少必要字段 %s", output.Name, field.Name)
			}

			if err := ValidateOutput(fieldValue, field); err != nil {
				return fmt.Errorf("对象 %s 的字段 %s 验证失败: %v", output.Name, field.Name, err)
			}
		}
	}
	return nil
}

// ParseOutput 解析单个输出
func ParseOutput(data any, output Output) (any, error) {
	if err := ValidateOutput(data, output); err != nil {
		return nil, err
	}

	switch output.Type {
	case "object":
		obj := data.(map[string]any)
		result := make(map[string]any)
		if len(output.Schema) == 0 {
			return obj, nil
		}
		for _, field := range output.Schema {
			fieldValue, exists := obj[field.Name]
			if !exists {
				// 使用默认值
				result[field.Name] = field.DeftValue
				continue
			}

			parsedValue, err := ParseOutput(fieldValue, field)
			if err != nil {
				return nil, err
			}
			result[field.Name] = parsedValue
		}
		return result, nil
	case "array":
		// 简单数组直接返回
		return data, nil
	default:
		// 简单类型直接返回
		return data, nil
	}
}

// ProcessNodeOutput 处理节点的所有输出
func ProcessNodeOutput(data map[string]any, outputs []Output) (map[string]any, error) {
	result := make(map[string]any)
	if len(data) == 0 {
		return nil, nil
	}

	for _, output := range outputs {
		value, exists := data[output.Name]
		if !exists {
			// 如果输入中没有对应的数据，使用默认值或零值代替
			if output.Type == "object" {
				r := make(map[string]any)
				err := json.Unmarshal([]byte(output.DeftValue.(string)), &r)
				if err != nil {
					// fmt.Printf("处理对象类输出 %s 的默认值时出错: %v\n", output.Name, err)
					result[output.Name] = r // 保持原有行为，即使解析失败也返回解析后的结果
				} else {
					result[output.Name] = r
				}
			} else {
				switch output.Type {
				case "integer":
					result[output.Name] = int64(0)
				case "float":
					result[output.Name] = float64(0)
				case "boolean":
					result[output.Name] = false
				case "array":
					result[output.Name] = make([]any, 0)
				case "object":
					result[output.Name] = make(map[string]any)
				case "string":
					result[output.Name] = ""
				default:
					result[output.Name] = output.DeftValue
				}
			}
			// fmt.Printf("使用默认值 %v 作为 %s 的输出\n", output.DeftValue, output.Name)
			continue
		}

		parsedValue, err := ParseOutput(value, output)
		if err != nil {
			return nil, fmt.Errorf("处理输出 %s 时出错: %v", output.Name, err)
		}

		result[output.Name] = parsedValue
		// fmt.Printf("输出 %s 已成功解析，类型: %s\n", output.Name, output.Type)
	}

	return result, nil
}
