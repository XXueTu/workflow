package components

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewGojaJsEngine(t *testing.T) {
	t.Logf("开始测试 GojaJsEngine...")

	// var jsScript = `
	// function main(params) {
	// 	var result = {
	// 		name: params.name,
	// 		age: params.age,
	// 		"md5": gmd5(params.name),
	// 	}
	// 	return result
	// }
	// `

	var jsScript = `
		function main(params) { const parsedParams = JSON.parse(params.info); var result = {name: parsedParams.name, age: parsedParams.age, "md5": gmd5(params.name), }; return result }
	`

	t.Logf("创建 GojaJsEngine...")
	engine, err := NewGojaJsEngine(jsScript, nil)
	if err != nil {
		t.Fatalf("创建引擎失败: %s\n", err)
	}
	t.Logf("引擎创建成功")

	t.Logf("准备执行 main 函数...")
	result, err := engine.Execute("main", map[string]interface{}{
		"info": `{"name": "xt", "age": 18}`,
		"name": "xuetu",
		"age":  18,
	})

	if err != nil {
		t.Fatalf("执行失败: %s\n", err)
	}
	t.Logf("执行成功，准备序列化结果...")

	marshal, err := json.Marshal(result)
	if err != nil {
		t.Fatal("序列化结果失败:", err)
	}

	t.Logf("测试完成，最终结果: %s\n", string(marshal))
	t.Logf("测试结果: %s\n", string(marshal))
	t.Logf("测试完成: %s\n", time.Now().Format("2006-01-02 15:04:05"))
}
