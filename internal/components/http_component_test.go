package components

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

func TestHTTPComponent_AnalyzeInputs(t *testing.T) {
	method := "http://localhost/{{block_output_100001.name}}/{{block_output_100002.name}}/sss/sss"
	// 解析有多少个{{}}
	re := regexp.MustCompile(`{{.*?}}`)
	matches := re.FindAllString(method, -1)
	blockMap := make(map[string]string, len(matches))
	// 解析{{}}
	for _, match := range matches {
		fmt.Printf("match: %v\n", match)
		// 去除{{}}
		blockName := strings.Trim(match, "{{}}")
		blockMap[match] = blockName
	}
	fmt.Printf("blockMap: %v\n", blockMap)

	// 替换表达式
	for key, value := range blockMap {
		method = strings.Replace(method, key, value, -1)
	}
	fmt.Printf("method: %v\n", method)
}
