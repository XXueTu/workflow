package components

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/XXueTu/workflow/internal/core"
)

var client *http.Client

// HTTPComponent HTTP请求组件
type HTTPComponent struct {
	config HTTPConfig
}

type HTTPConfig struct {
	URL     string        `json:"url"`
	Method  string        `json:"method"`
	Headers []core.Inputs `json:"headers"`
	Params  []core.Inputs `json:"params"`
	Body    []core.Inputs `json:"body,omitempty"`
	Retries int           `json:"retries"`
	Timeout int64         `json:"timeout"`
}

func NewHTTPComponent(config json.RawMessage) (*HTTPComponent, error) {
	c := HTTPConfig{}
	if err := json.Unmarshal(config, &c); err != nil {
		return nil, fmt.Errorf("解析HTTP配置失败: %v", err)
	}
	if c.Timeout == 0 {
		c.Timeout = 10
	}
	if c.Retries == 0 {
		c.Retries = 1
	}
	timeout := time.Duration(c.Timeout) * time.Second
	c.Timeout = int64(timeout)
	return &HTTPComponent{config: c}, nil
}

func (c *HTTPComponent) Validate() []core.ValidationError {
	var errors []core.ValidationError
	if c.config.URL == "" {
		errors = append(errors, core.ValidationError{
			Field:   "url",
			Message: "URL不能为空",
		})
	}
	if c.config.Method == "" {
		errors = append(errors, core.ValidationError{
			Field:   "method",
			Message: "请求方法不能为空",
		})
	}
	return errors
}

var headerKey = "headers"
var bodyKey = "body"
var urlKey = "url"
var isJSONKey = "isJSON"

func (c *HTTPComponent) AnalyzeInputs(ctx context.Context) (any, error) {
	var input map[string]any = make(map[string]any, 3)
	execCtx := ctx.(*core.ExecutionContext)
	headers, err := core.ParseNodeInputs(c.config.Headers, execCtx)
	if err != nil {
		return nil, err
	}
	headersMap := make(map[string]string)
	for k, v := range headers {
		headersMap[k] = fmt.Sprintf("%v", v)
	}
	input[headerKey] = headersMap

	// body
	params := make(map[string]any)
	isJSON := false
	if len(c.config.Params) != 0 {
		params, err = core.ParseNodeInputs(c.config.Params, execCtx)
		if err != nil {
			fmt.Printf("解析params失败: %v\n", err)
			return nil, err
		}
	}
	if len(c.config.Body) != 0 {
		params, err = core.ParseNodeInputs(c.config.Body, execCtx)
		if err != nil {
			fmt.Printf("解析body失败: %v\n", err)
			return nil, err
		}
		isJSON = true
	}
	// method http://localhost/{{block_output_100001.name}}/sss/sss 表达式{{}}如何解析
	url, err := parseMethod(execCtx, c.config.URL)
	if err != nil {
		return nil, err
	}
	input[urlKey] = url
	input[bodyKey] = params
	input[isJSONKey] = isJSON

	return input, nil
}

func parseMethod(execCtx *core.ExecutionContext, method string) (string, error) {
	fmt.Printf("method before: %v\n", method)
	// 解析有多少个{{}}
	re := regexp.MustCompile(`{{.*?}}`)
	matches := re.FindAllString(method, -1)
	blockMap := make(map[string]string, len(matches))
	// 解析{{}}
	for _, match := range matches {
		fmt.Printf("match: %v\n", match)
		// 去除{{}}
		blockName := strings.Trim(match, "{{}}")
		// start-node-1.output.name 读取截取output前的内容
		blockNames := strings.Split(blockName, ".")
		if len(blockNames) > 1 {
			blockName = blockNames[0] + "." + blockNames[1]
		}
		value, ok := execCtx.GetVariable(blockName)
		if !ok {
			return "", fmt.Errorf("找不到变量: %v", blockName)
		}

		for i := 2; i < len(blockNames); i++ {
			// 嵌套查找 可能存在info.city
			value, ok = value.(map[string]any)[blockNames[i]]
			if !ok {
				return "", fmt.Errorf("找不到变量: %v,all:%s", blockName, match)
			}
		}
		blockMap[match] = fmt.Sprintf("%s", value)
	}
	fmt.Printf("blockMap: %+v\n", blockMap)
	// 替换表达式
	for key, value := range blockMap {
		method = strings.Replace(method, key, value, -1)
	}
	fmt.Printf("method after: %v\n", method)
	return method, nil
}

func (c *HTTPComponent) Execute(ctx context.Context, input any) (*core.Result, error) {
	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("input 类型不匹配")
	}
	headersMap, ok := inputMap[headerKey].(map[string]string)
	if !ok {
		return nil, fmt.Errorf("headersMap 类型不匹配")
	}
	params, ok := inputMap[bodyKey].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("params 类型不匹配")
	}
	isJSON, ok := inputMap[isJSONKey].(bool)
	if !ok {
		return nil, fmt.Errorf("isJSON 类型不匹配")
	}
	url, ok := inputMap[urlKey].(string)
	if !ok {
		return nil, fmt.Errorf("url 类型不匹配")
	}

	client := NewHttpClient(time.Duration(c.config.Timeout), c.config.Retries)
	statusCode, body, err := client.DoRequest(RequestOptions{
		Method:  c.config.Method,
		URL:     url,
		Headers: headersMap,
		Body:    params,
		IsJSON:  isJSON,
	})
	if err != nil {
		fmt.Printf("http request err:%s", err.Error())
		return &core.Result{
			Route:  []string{Failed},
			Output: nil,
		}, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return &core.Result{
			Route:  []string{Failed},
			Output: nil,
		}, err
	}
	jsonHeaders, _ := json.Marshal(headersMap)
	r := map[string]any{
		"body":       string(body),
		"statusCode": statusCode,
		"headers":    string(jsonHeaders),
	}
	return &core.Result{
		Output: r,
		Route:  []string{Success},
	}, nil
}

// 复用单例 fasthttp.Client
var globalClient = &fasthttp.Client{}

// 使用 sync.Pool 进行 Request/Response 复用
var requestPool = sync.Pool{
	New: func() interface{} {
		return &fasthttp.Request{}
	},
}

var responsePool = sync.Pool{
	New: func() interface{} {
		return &fasthttp.Response{}
	},
}

type HttpClient struct {
	client     *fasthttp.Client
	Timeout    time.Duration
	MaxRetries int
}

// NewHttpClient 复用单例 client
func NewHttpClient(timeout time.Duration, maxRetries int) *HttpClient {
	return &HttpClient{
		client:     globalClient,
		Timeout:    timeout,
		MaxRetries: maxRetries,
	}
}

// RequestOptions 请求选项
type RequestOptions struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    interface{}
	IsJSON  bool
}

// DoRequest 复用 Request/Response
func (h *HttpClient) DoRequest(opts RequestOptions) (int, []byte, error) {
	req := requestPool.Get().(*fasthttp.Request)
	resp := responsePool.Get().(*fasthttp.Response)
	defer requestPool.Put(req)
	defer responsePool.Put(resp)

	req.Reset()
	resp.Reset()

	req.SetRequestURI(opts.URL)
	req.Header.SetMethod(opts.Method)

	for key, value := range opts.Headers {
		req.Header.Set(key, value)
	}

	if opts.Body != nil {
		var bodyBytes []byte
		var err error
		if opts.IsJSON {
			bodyBytes, err = json.Marshal(opts.Body)
			if err != nil {
				return 0, nil, err
			}
			req.Header.Set("Content-Type", "application/json")
		} else {
			if str, ok := opts.Body.(string); ok {
				bodyBytes = []byte(str)
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			} else if formData, ok := opts.Body.(map[string]string); ok {
				var formDataBytes bytes.Buffer
				for key, value := range formData {
					_, _ = formDataBytes.WriteString(key + "=" + value + "&")
				}
				bodyBytes = formDataBytes.Bytes()[:len(formDataBytes.Bytes())-1] // 去掉最后一个&
				req.Header.Set("Content-Type", "multipart/form-data")
			} else {
				return 0, nil, errors.New("invalid form body format")
			}
		}
		req.SetBody(bodyBytes)
	}

	for i := 0; i <= h.MaxRetries; i++ {
		err := h.client.DoTimeout(req, resp, h.Timeout)
		if err == nil {
			return resp.StatusCode(), resp.Body(), nil
		}
		fmt.Printf("http request err:%s,retry:%d\n", err.Error(), i+1)
	}

	return 0, nil, errors.New("request failed after retries")
}
