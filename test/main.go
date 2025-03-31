package main

import (
	"fmt"

	"github.com/tidwall/gjson"
)

func main() {
	inputJSON := `{"code":"0","message":"success","users":[{"age":57,"gender":"女","name":"郗拜串","phone":"14629****8768"},{"age":44,"gender":"女","name":"高衡三","phone":"13693****0751"},{"age":104,"gender":"女","name":"司城衷","phone":"17184****7574"},{"age":37,"gender":"女","name":"奚瞪绳","phone":"15199****7812"},{"age":7,"gender":"男","name":"牧横","phone":"10133****4613"}]}`

	// 使用 gjson 的过滤器语法
	matches := gjson.Get(inputJSON, `users.#(age>=80)`)
	fmt.Println(matches.String())
}
