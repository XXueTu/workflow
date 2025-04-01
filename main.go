package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/XXueTu/workflow/pkg/core"
	"github.com/XXueTu/workflow/pkg/engine"
	"github.com/XXueTu/workflow/pkg/metrics"
)

func main() {
	// 创建工作流引擎
	eg := engine.NewWorkflowEngine()
	defer eg.Cleanup() // 确保在程序结束时释放资源
	metrics.NewCollector()

	// workflow, err := os.ReadFile("start.json")
	workflow, err := os.ReadFile("start_http.json")
	// workflow, err := os.ReadFile("start_http_parallel.json")
	// workflow, err := os.ReadFile("start_js.json")
	// workflow, err := os.ReadFile("start_branch.json")
	if err != nil {
		log.Fatalf("读取工作流文件失败: %v", err)
	}
	workflowDefinition := core.WorkflowDef{}
	err = json.Unmarshal(workflow, &workflowDefinition)
	if err != nil {
		log.Fatalf("解析工作流文件失败: %v", err)
	}

	definitionBytes, _ := json.MarshalIndent(workflowDefinition, "", "  ")
	log.Printf("工作流定义: %s", string(definitionBytes))

	// 创建带有超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 注册工作流
	err = eg.RegisterWorkflow(ctx, &workflowDefinition)
	if err != nil {
		log.Fatalf("工作流注册失败: %v", err)
	}

	param := `
{
	"name": "cvl313pnhrgp7l45jlk0",
	"age": 18,
	"age1": 18,
	"info": {
		"city": "hangzhou",
		"country": "china",
		"height": 170.5,
		"weight": 65.5,
		"phone": 17635800128,
		"email": ["zhangsan@example.com", "lisi@example.com"],
		"like": {
			"food": ["apple", "banana"],
			"sport": ["basketball", "football"]
		}
	},
	"address": ["beijing", "shanghai","hangzhou"],
	"mritalStatus":false,
	"sight":1.0
}`
	mapParam := make(map[string]any)
	err = json.Unmarshal([]byte(param), &mapParam)
	if err != nil {
		log.Fatalf("解析参数失败: %v", err)
	}
	// 执行工作流
	if err := eg.ExecuteWorkflow(ctx, workflowDefinition.ID, "serial_123", mapParam); err != nil {
		log.Fatalf("工作流执行失败: %v", err)
	}

	endResult, ok := eg.GetNodeResult(workflowDefinition.ID, "serial_123", "end-node-1")
	if !ok {
		log.Printf("未找到 end-node-1 节点的执行结果")
		return
	}
	endResultBytes, _ := json.MarshalIndent(endResult, "", "  ")
	log.Printf("工作流执行结果:\n%s", string(endResultBytes))

	httpResult, ok := eg.GetNodeResult(workflowDefinition.ID, "serial_123", "http-node-1")
	if !ok {
		log.Printf("未找到 http-node-1 节点的执行结果")
		return
	}
	httpResultBytes, _ := json.MarshalIndent(httpResult, "", "  ")
	log.Printf("http:工作流执行结果:\n%s", string(httpResultBytes))

	// // 执行工作流并统计时间
	// startTime := time.Now()
	// if err := eg.ExecuteWorkflow(ctx, workflow.ID, "serial_123", map[string]interface{}{}); err != nil {
	// 	log.Fatalf("工作流执行失败: %v", err)
	// }
	// endTime := time.Now()
	// executionTime := endTime.Sub(startTime)
	// log.Printf("工作流执行时间: %v", executionTime)

	// // 获取 filter-users 节点的结果
	// result, ok := eg.GetNodeResult(workflow.ID, "serial_123", "filter-users")
	// if !ok {
	// 	log.Printf("未找到 filter-users 节点的执行结果")
	// 	return
	// }

	// // 打印执行结果
	// resultBytes, _ := json.MarshalIndent(result.Data, "", "  ")
	// log.Printf("工作流执行结果:\n%s", string(resultBytes))
}
