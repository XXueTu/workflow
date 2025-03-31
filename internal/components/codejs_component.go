package components

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"

	"github.com/XXueTu/workflow/flow-engine/internal/core"
)

// CodejsComponent 代码执行组件
type CodejsComponent struct {
	engine *GojaJsEngine
}

var codejsComponentPool = sync.Pool{
	New: func() interface{} {
		return &CodejsComponent{}
	},
}

type CodejsConfig struct {
	Code string `json:"code"`
}

func NewCodejsComponent(config json.RawMessage) (*CodejsComponent, error) {
	c := codejsComponentPool.Get().(*CodejsComponent)
	var codejsConfig CodejsConfig
	if err := json.Unmarshal(config, &codejsConfig); err != nil {
		return nil, fmt.Errorf("解析代码执行组件配置失败: %v", err)
	}
	fmt.Printf("codejsConfig.Code: %s\n", codejsConfig.Code)
	engine, err := NewGojaJsEngine(codejsConfig.Code, nil)
	if err != nil {
		fmt.Printf("创建代码执行组件失败: %v\n", err)
		return nil, fmt.Errorf("创建代码执行组件失败: %v", err)
	}
	c.engine = engine
	return c, nil
}

func (c *CodejsComponent) Execute(ctx context.Context, input any) (*core.Result, error) {

	result, err := c.engine.Execute("main", input)
	if err != nil {
		return nil, errors.New("执行代码执行组件失败: " + err.Error())
	}

	return &core.Result{
		Output: result,
		Route:  []string{Success},
	}, nil
}

func (c *CodejsComponent) Validate() []core.ValidationError {
	return nil
}

func (c *CodejsComponent) AnalyzeInputs(ctx context.Context) (any, error) {
	return nil, nil
}

// GojaJsEngine goja js engine
type GojaJsEngine struct {
	vmPool            sync.Pool
	config            Config
	jsScript          *goja.Program
	jsUdfProgramCache map[string]*goja.Program
}

type Config struct {
	Udf map[string]interface{}
}

// NewGojaJsEngine Create a new instance of the JavaScript engine
func NewGojaJsEngine(jsScript string, fromVars map[string]interface{}) (*GojaJsEngine, error) {
	config := Config{
		Udf: make(map[string]interface{}),
	}
	if config.Udf == nil {
		config.Udf = make(map[string]interface{})
	}
	name := "gmd5"
	config.Udf[name] = func(v string) string {
		return fmt.Sprintf("%x", md5.Sum([]byte(v)))
	}

	program, err := goja.Compile("", jsScript, true)
	if err != nil {
		return nil, err
	}
	jsEngine := &GojaJsEngine{
		config:   config,
		jsScript: program,
	}
	if err = jsEngine.PreCompileJs(config); err != nil {
		return nil, err
	}
	jsEngine.vmPool = sync.Pool{
		New: func() interface{} {
			return jsEngine.NewVm(config, fromVars)
		},
	}
	return jsEngine, nil
}

// PreCompileJs Precompiled UDF JavaScript file
func (g *GojaJsEngine) PreCompileJs(config Config) error {
	var jsUdfProgramCache = make(map[string]*goja.Program)
	for k, v := range config.Udf {
		if jsFuncStr, ok := v.(string); ok {
			if p, err := goja.Compile(k, jsFuncStr, true); err != nil {
				return err
			} else {
				jsUdfProgramCache[k] = p
			}
		} else if script, scriptOk := v.(Script); scriptOk {
			if script.Type == Js || script.Type == "" {
				if c, ok := script.Content.(string); ok {
					if p, err := goja.Compile(k, c, true); err != nil {
						return err
					} else {
						jsUdfProgramCache[k] = p
					}
				} else if p, ok := script.Content.(*goja.Program); ok {
					jsUdfProgramCache[k] = p
				}
			}
		}
	}
	g.jsUdfProgramCache = jsUdfProgramCache

	return nil
}

// NewVm new a js VM
func (g *GojaJsEngine) NewVm(config Config, fromVars map[string]interface{}) *goja.Runtime {
	vm := goja.New()
	vars := make(map[string]interface{})
	if fromVars != nil {
		for k, v := range fromVars {
			vars[k] = v
		}
	}

	// Add global custom functions to the JavaScript runtime
	for k, v := range config.Udf {
		var err error
		if _, ok := v.(string); ok {
			if p, ok := g.jsUdfProgramCache[k]; ok {
				_, err = vm.RunProgram(p)
			}
		} else if script, scriptOk := v.(Script); scriptOk {
			if script.Type == Js || script.Type == "" {
				// parse  JS script
				if _, ok := script.Content.(string); ok {
					if p, ok := g.jsUdfProgramCache[k]; ok {
						_, err = vm.RunProgram(p)
					}
				} else if _, ok := script.Content.(*goja.Program); ok {
					if p, ok := g.jsUdfProgramCache[k]; ok {
						_, err = vm.RunProgram(p)
					}
				} else {
					funcName := strings.Replace(k, Js+"#", "", 1)
					vars[funcName] = vm.ToValue(script.Content)
				}
			}
		} else {
			// parse go func
			vars[k] = vm.ToValue(v)
		}
		if err != nil {
			fmt.Println("parse js script=" + k + " error,err:" + err.Error())
		}
	}
	for k, v := range vars {
		if err := vm.Set(k, v); err != nil {
			fmt.Println("set variable error,err:" + err.Error())
		}
	}

	state := g.setTimeout(vm)

	_, err := vm.RunProgram(g.jsScript)
	// If there is no timeout, state=0; otherwise, state=-2
	closeStateChan(state)

	if err != nil {
		fmt.Println("js vm error,err:" + err.Error())
	}
	return vm
}

// Execute Execute JavaScript script
func (g *GojaJsEngine) Execute(functionName string, argumentList ...interface{}) (out interface{}, err error) {
	defer func() {
		if caught := recover(); caught != nil {
			err = fmt.Errorf("%s", caught)
		}
	}()

	vm := g.vmPool.Get().(*goja.Runtime)

	// vm.Set(CtxKey, ctx)

	state := g.setTimeout(vm)

	f, ok := goja.AssertFunction(vm.Get(functionName))
	if !ok {
		return nil, errors.New(functionName + " is not a function")
	}
	var params []goja.Value
	for _, v := range argumentList {
		params = append(params, vm.ToValue(v))
	}
	res, err := f(goja.Undefined(), params...)
	// If there is no timeout, state=0; otherwise, state=-2
	closeStateChan(state)
	// Put back to the pool
	g.vmPool.Put(vm)
	if err != nil {
		return nil, err
	}
	return res.Export(), err
}

func (g *GojaJsEngine) Stop() {
}

// setTimeout if timeout interrupt the js script execution
func (g *GojaJsEngine) setTimeout(vm *goja.Runtime) chan int {
	state := make(chan int, 1)
	state <- 0
	time.AfterFunc(5*time.Second, func() {
		if <-state == 0 {
			state <- 2
			vm.Interrupt("execution timeout")
		}
	})
	return state
}

func closeStateChan(state chan int) {
	if <-state == 0 {
		state <- 1
	}
	close(state)
}

// Script is used to register native functions or custom functions defined in Go.
type Script struct {
	// Type is the script type, default is Js.
	Type string
	// Content is the script content or custom function.
	Content interface{}
}

const (
	Js = "Js" // Represents JavaScript scripting language.
)
