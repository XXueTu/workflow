package engine

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/ants/v2"
	"golang.org/x/sync/errgroup"

	"github.com/XXueTu/workflow/pkg/components"
	"github.com/XXueTu/workflow/pkg/core"
	"github.com/XXueTu/workflow/pkg/observability"
)

// WorkflowEngine 工作流引擎
type WorkflowEngine struct {
	// 核心组件
	pool     *ants.PoolWithFunc   // 任务执行池
	poolSize int                  // 当前池大小
	tracer   observability.Tracer // 追踪器

	// 配置
	config *EngineConfig // 引擎配置

	// 工作流管理
	executorPool map[string]*Executor // 工作流执行器池
	mu           sync.RWMutex         // 读写锁

	// 资源管理
	cleanup    chan string   // 清理通道
	shutdownCh chan struct{} // 关闭信号通道

	// 监控指标
	metricsCollector *MetricsCollector // 指标收集器
}

// NewWorkflowEngine 创建新的工作流引擎
func NewWorkflowEngine(opts ...Option) *WorkflowEngine {
	config := DefaultConfig()
	engine := &WorkflowEngine{
		tracer:       observability.NewTracer(),
		executorPool: make(map[string]*Executor),
		config:       config,
		cleanup:      make(chan string, 100),
		shutdownCh:   make(chan struct{}),
		metricsCollector: &MetricsCollector{
			metrics: make(map[string]*core.WorkflowMetrics),
		},
	}

	for _, opt := range opts {
		opt(engine)
	}

	// 创建任务处理函数
	taskHandler := func(i any) {
		task, ok := i.(*ExecutionTask)
		if !ok {
			return
		}
		defer task.Sw.Done()
		result, err := task.Engine.executeNode(task.Context, task.NodeID, task.Executor.definition.Nodes[task.NodeID], task.Component)
		if err != nil {
			fmt.Printf("执行节点 [%s] 失败: %v\n", task.NodeID, err)
			task.Context.SetError(task.NodeID, err)
			return
		}
		fmt.Printf("执行节点 [%s] 结果: %v\n", task.NodeID, result.Output)
	}

	// 创建 ants 函数池
	pool, err := ants.NewPoolWithFunc(engine.config.InitialPoolSize, taskHandler,
		ants.WithPreAlloc(true),
		ants.WithMaxBlockingTasks(engine.config.MaxConcurrentWorkflows),
		ants.WithNonblocking(true),
	)
	if err != nil {
		panic(err)
	}
	engine.pool = pool
	engine.poolSize = engine.config.InitialPoolSize

	// 启动清理和指标收集协程
	engine.startCleanupRoutine()
	if engine.config.EnableMetrics {
		engine.startMetricsCollector()
	}

	return engine
}

// RegisterWorkflow 注册工作流
func (e *WorkflowEngine) RegisterWorkflow(ctx context.Context, def *core.WorkflowDef) error {

	// 检查并处理已存在的工作流
	if err := e.handleExistingWorkflow(def.ID); err != nil {
		return err
	}

	// 构建执行计划
	plan, err := e.buildExecutionPlan(ctx, def)
	if err != nil {
		return fmt.Errorf("无法构建执行计划: %v", err)
	}

	// 构建条件路由
	condition, err := e.buildConditionRouter(def)
	if err != nil {
		return fmt.Errorf("无法构建条件映射: %v", err)
	}

	// 创建新的执行器
	executor := e.createExecutor(def, plan, condition)

	// 注册执行器
	e.registerExecutor(def.ID, executor)

	return nil
}

// handleExistingWorkflow 处理已存在的工作流
func (e *WorkflowEngine) handleExistingWorkflow(workflowID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if existing, exists := e.executorPool[workflowID]; exists {
		atomic.StoreInt64(&existing.lastAccessed, time.Now().UnixNano())
		existing.status = core.WorkflowStatusDeploying
	}
	return nil
}

// createExecutor 创建执行器
func (e *WorkflowEngine) createExecutor(def *core.WorkflowDef, plan *ExecutionPlan, condition map[string][]string) *Executor {
	return &Executor{
		definition:      def,
		env:             make(map[string]any),
		executionPlan:   plan,
		conditionRouter: condition,
		totalNodes:      int64(len(def.Nodes)),
		status:          core.WorkflowStatusActive,
		createdAt:       time.Now(),
		defaultTTL:      e.config.DefaultContextTTL,
		shutdownCh:      make(chan struct{}),
		metrics:         &core.WorkflowMetrics{LastExecutionTime: time.Now()},
	}
}

// registerExecutor 注册执行器
func (e *WorkflowEngine) registerExecutor(workflowID string, executor *Executor) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.executorPool[workflowID] = executor

	if e.config.EnableMetrics {
		e.metricsCollector.mu.Lock()
		e.metricsCollector.metrics[workflowID] = executor.metrics
		e.metricsCollector.mu.Unlock()
	}
}

// DeregisterWorkflow 注销工作流
func (e *WorkflowEngine) DeregisterWorkflow(ctx context.Context, workflowID string, graceful bool) error {
	e.mu.Lock()
	executor, exists := e.executorPool[workflowID]
	if !exists {
		e.mu.Unlock()
		return fmt.Errorf("工作流未找到: %s", workflowID)
	}

	executor.status = core.WorkflowStatusShutdown
	e.mu.Unlock()

	if !graceful {
		return e.immediateDeregister(workflowID, executor)
	}

	return e.gracefulDeregister(ctx, workflowID, executor)
}

// immediateDeregister 立即注销工作流
func (e *WorkflowEngine) immediateDeregister(workflowID string, executor *Executor) error {
	e.mu.Lock()
	delete(e.executorPool, workflowID)
	e.mu.Unlock()
	close(executor.shutdownCh)
	return nil
}

// gracefulDeregister 优雅注销工作流
func (e *WorkflowEngine) gracefulDeregister(ctx context.Context, workflowID string, executor *Executor) error {
	go func() {
		timeout := time.After(30 * time.Second)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if atomic.LoadInt64(&executor.activeCount) == 0 {
					e.immediateDeregister(workflowID, executor)
					return
				}
			case <-timeout:
				e.immediateDeregister(workflowID, executor)
				return
			case <-ctx.Done():
				e.immediateDeregister(workflowID, executor)
				return
			}
		}
	}()

	return nil
}

// ExecuteWorkflow 执行工作流
func (e *WorkflowEngine) ExecuteWorkflow(ctx context.Context, workflowID string, serialID string, params map[string]any) error {
	// 参数校验
	if params == nil {
		return fmt.Errorf("参数为空")
	}

	// 获取工作流执行器
	e.mu.RLock()
	executor, ok := e.executorPool[workflowID]
	e.mu.RUnlock()

	if !ok {
		return fmt.Errorf("工作流未找到: %s", workflowID)
	}

	// 检查工作流状态
	if executor.status == core.WorkflowStatusShutdown {
		return fmt.Errorf("工作流正在关闭: %s", workflowID)
	}

	// 更新访问时间
	atomic.StoreInt64(&executor.lastAccessed, time.Now().UnixNano())

	// 开始追踪
	var span observability.Span
	if e.config.EnableTracing {
		span = e.tracer.StartSpan(ctx, "workflow_execution")
		defer span.End()
	}

	// 创建带超时的上下文
	execCtx, cancel := context.WithTimeout(ctx, e.config.ExecutionTimeout)

	// 创建执行上下文
	executionContext := core.NewExecutionContext(execCtx, serialID, executor.totalNodes, params)
	executionContext.Context = execCtx
	executionContext.Expiration = time.Now().Add(executor.defaultTTL)

	// 存储取消函数到执行上下文中，以便在暂停时可以取消
	executionContext.SetVariable("cancel", cancel)

	// 设置初始状态为运行中
	executionContext.State.Status = core.StatusRunning

	// 存储执行上下文（使用sync.Map避免锁争用）
	executor.execContexts.Store(serialID, executionContext)

	// 增加活跃执行计数
	atomic.AddInt64(&executor.activeCount, 1)
	if e.config.EnableMetrics {
		atomic.AddInt64(&executor.metrics.ActiveExecutions, 1)
	}

	// 使用errgroup进行错误处理
	startTime := time.Now()

	// 执行工作流
	err := e.executeWorkflowPhases(execCtx, executor, executionContext)

	// 记录执行时间
	executionDuration := time.Since(startTime)

	// 减少活跃执行计数
	atomic.AddInt64(&executor.activeCount, -1)

	// 更新指标
	if e.config.EnableMetrics {
		atomic.AddInt64(&executor.metrics.ActiveExecutions, -1)

		if err != nil {
			atomic.AddInt64(&executor.metrics.FailedExecutions, 1)
		} else {
			atomic.AddInt64(&executor.metrics.CompletedExecutions, 1)

			// 更新平均执行时间
			oldAvg := executor.metrics.AverageExecutionTime
			completed := float64(atomic.LoadInt64(&executor.metrics.CompletedExecutions))
			newAvg := oldAvg + (float64(executionDuration.Milliseconds())-oldAvg)/completed
			executor.metrics.AverageExecutionTime = newAvg
		}
		executor.metrics.LastExecutionTime = time.Now()
	}

	// 处理执行结果
	if err != nil {
		executionContext.State.Status = core.StatusFailed
		executionContext.SetError("workflow", err)
		// 确保调用取消函数
		cancel()
		return err
	}

	executionContext.State.Status = core.StatusCompleted
	// 确保调用取消函数
	cancel()
	return nil
}

// executeWorkflowPhases 执行工作流阶段
func (e *WorkflowEngine) executeWorkflowPhases(ctx context.Context, executor *Executor, execCtx *core.ExecutionContext) error {
	// 总执行计划
	fmt.Printf("总执行计划:%d\n", len(executor.executionPlan.Phases))
	for phaseIdx, phase := range executor.executionPlan.Phases {
		if err := e.executePhase(ctx, executor, execCtx, phase, phaseIdx); err != nil {
			return err
		}
	}
	return nil
}

// executePhase 执行单个阶段
func (e *WorkflowEngine) executePhase(ctx context.Context, executor *Executor, execCtx *core.ExecutionContext, phase ExecutionPhase, phaseIdx int) error {
	var sw sync.WaitGroup
	nodes := make([]string, len(phase.Nodes))
	g, ctx := errgroup.WithContext(ctx)
	fmt.Printf("执行阶段:%d,节点数:%d\n", phaseIdx, len(phase.Nodes))
	for i, node := range phase.Nodes {
		if err := e.submitNodeTask(ctx, executor, execCtx, node, &sw, g, nodes, i); err != nil {
			return err
		}
	}

	if err := g.Wait(); err != nil {
		return err
	}
	sw.Wait()

	return nil
}

// submitNodeTask 提交节点任务
func (e *WorkflowEngine) submitNodeTask(ctx context.Context, executor *Executor, execCtx *core.ExecutionContext, node *WorkflowNode, sw *sync.WaitGroup, g *errgroup.Group, nodes []string, index int) error {
	fmt.Printf("执行节点: %s\n", node.ID)
	nodes[index] = node.ID

	// 检查上下文状态
	if err := e.checkContext(ctx, executor); err != nil {
		return err
	}

	// 获取节点定义副本
	nodeCopy := node

	// 增加等待计数
	sw.Add(1)

	// 提交任务
	g.Go(func() error {
		if err := e.handleNodeExecution(execCtx, executor, nodeCopy, sw); err != nil {
			return err
		}
		return nil
	})

	return nil
}

// checkContext 检查上下文状态
func (e *WorkflowEngine) checkContext(ctx context.Context, executor *Executor) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-executor.shutdownCh:
		return fmt.Errorf("工作流正在关闭")
	default:
		return nil
	}
}

// handleNodeExecution 处理节点执行
func (e *WorkflowEngine) handleNodeExecution(execCtx *core.ExecutionContext, executor *Executor, node *WorkflowNode, sw *sync.WaitGroup) error {
	// 开始类型的组件跳过路由检查
	if node.Type == components.Start || node.Type == components.StartItem {
		sw.Done()
		return e.executeStartNode(execCtx, node)
	}

	// 检查路由
	if err := e.checkNodeRoute(execCtx, executor, node); err != nil {
		return err
	}

	// 创建并执行组件
	component, err := components.ComponentFactory(e, node.Type, node.NodeDefinition)
	if err != nil {
		return fmt.Errorf("无法创建组件 [%s]: %v", node.ID, err)
	}

	task := &ExecutionTask{
		NodeID:     node.ID,
		Component:  component,
		Context:    execCtx,
		Engine:     e,
		Executor:   executor,
		WorkflowID: executor.definition.ID,
		SerialID:   execCtx.Id,
		Sw:         sw,
	}

	return e.pool.Invoke(task)
}

// executeStartNode 执行开始节点
func (e *WorkflowEngine) executeStartNode(execCtx *core.ExecutionContext, node *WorkflowNode) error {
	params, bool := execCtx.GetVariable("_zero")
	if !bool {
		return fmt.Errorf("无数据输入")
	}
	output, err := core.ProcessNodeOutput(params.(map[string]any), node.Outputs)
	if err != nil {
		return err
	}

	execCtx.SetVariable(node.ID+".output", output)
	nodeResult := &core.NodeResult{
		Input:    "",
		Output:   output,
		Route:    []string{components.Success},
		NodeID:   node.ID,
		Duration: 0,
		Error:    "",
		Type:     node.Type,
	}
	return e.updateWorkflowState(execCtx, []*core.NodeResult{nodeResult})
}

// checkNodeRoute 检查节点路由
func (e *WorkflowEngine) checkNodeRoute(execCtx *core.ExecutionContext, executor *Executor, node *WorkflowNode) error {
	route, ok := executor.conditionRouter[node.ID]
	if !ok {
		return fmt.Errorf("节点初始化路由未找到: %s", node.ID)
	}

	if !execCtx.CheckRoute(route) {
		fmt.Printf("路由:%+v\n", route)
		return e.handleSkippedNode(execCtx, node)
	}

	return nil
}

// handleSkippedNode 处理跳过的节点
func (e *WorkflowEngine) handleSkippedNode(execCtx *core.ExecutionContext, node *WorkflowNode) error {
	fmt.Printf("%s节点必要路由未找到,跳过\n", node.ID)

	output, err := core.ProcessNodeOutput(map[string]any{}, node.Outputs)
	if err != nil {
		return err
	}

	execCtx.SetVariable(node.ID+".output", output)

	nodeResult := &core.NodeResult{
		Input:    map[string]any{},
		Output:   output,
		Route:    []string{components.Skip},
		NodeID:   node.ID,
		Duration: 0,
		Error:    "",
		Type:     node.Type,
	}

	return e.updateWorkflowState(execCtx, []*core.NodeResult{nodeResult})
}

// prepareNodeInput 准备节点输入数据
func (e *WorkflowEngine) prepareNodeInput(ctx *core.ExecutionContext, node *core.NodeDefinition, component components.Component) (any, error) {
	if node.Type == "start" {
		zero, ok := ctx.GetVariable("_zero")
		if !ok {
			return nil, fmt.Errorf("输入参数未找到")
		}
		return core.ProcessNodeOutput(zero.(map[string]any), node.Outputs)
	}

	// 初始化合并结果
	mergedInput := make(map[string]any)

	// 1. 获取自定义输入
	customInput, err := component.AnalyzeInputs(ctx)
	if err != nil {
		return nil, fmt.Errorf("分析自定义输入失败: %v", err)
	}

	// 2. 如果有自定义输入，先复制到合并结果中
	if customInput != nil {
		if customMap, ok := customInput.(map[string]any); ok {
			for k, v := range customMap {
				mergedInput[k] = v
			}
		} else {
			return nil, fmt.Errorf("自定义输入必须是map类型")
		}
	}

	// 3. 获取标准输入
	standardInput, err := core.ParseNodeInputs(node.Inputs, ctx)
	if err != nil {
		return nil, fmt.Errorf("解析标准输入失败: %v", err)
	}

	// 4. 合并标准输入
	maps.Copy(mergedInput, standardInput)

	// 5. 如果没有任何输入，返回空map
	if len(mergedInput) == 0 {
		return map[string]any{}, nil
	}

	return mergedInput, nil
}

// processNodeOutput 处理节点输出数据
func (e *WorkflowEngine) processNodeOutput(input any, result *core.Result, node *core.NodeDefinition) (any, error) {
	if node.Type == components.End || node.Type == components.EndItem {
		return input, nil
	}

	if result.Output == nil {
		return map[string]any{}, nil
	}

	output, err := core.ProcessNodeOutput(result.Output.(map[string]any), node.Outputs)
	if err != nil {
		return nil, fmt.Errorf("处理输出映射失败: %v", err)
	}
	return output, nil
}

// executeNode 执行节点
func (e *WorkflowEngine) executeNode(ctx *core.ExecutionContext, nodeID string, node *core.NodeDefinition, component components.Component) (*core.Result, error) {

	startTime := time.Now()

	// 1. 验证组件
	if err := e.validateComponent(component, node); err != nil {
		return nil, e.handleNodeError(ctx, node, err, startTime)
	}

	// 2. 准备输入数据
	input, err := e.prepareNodeInput(ctx, node, component)
	if err != nil {
		return nil, e.handleNodeError(ctx, node, err, startTime)
	}

	// 3. 执行组件
	result, err := e.executeComponent(ctx, component, input)
	if err != nil {
		return nil, e.handleNodeError(ctx, node, err, startTime)
	}

	// 4. 处理输出数据
	output, err := e.processNodeOutput(input, result, node)
	if err != nil {
		return nil, e.handleNodeError(ctx, node, err, startTime)
	}

	// 5. 更新上下文
	if err := e.updateNodeContext(ctx, nodeID, output); err != nil {
		return nil, e.handleNodeError(ctx, node, err, startTime)
	}

	// 6. 创建并返回结果
	nodeResult := e.createNodeResult(node, input, output, result, startTime)
	if err := e.updateWorkflowState(ctx, []*core.NodeResult{nodeResult}); err != nil {
		fmt.Printf("更新工作流状态失败: %v\n", err)
	}

	return result, nil
}

// validateComponent 验证组件
func (e *WorkflowEngine) validateComponent(component components.Component, node *core.NodeDefinition) error {
	if validateErrors := component.Validate(); len(validateErrors) > 0 {
		return fmt.Errorf("组件 [%s] 验证失败: %v", node.ID, validateErrors)
	}
	return nil
}

// executeComponent 执行组件
func (e *WorkflowEngine) executeComponent(ctx *core.ExecutionContext, component components.Component, input any) (*core.Result, error) {
	return component.Execute(ctx, input)
}

// updateNodeContext 更新节点上下文
func (e *WorkflowEngine) updateNodeContext(ctx *core.ExecutionContext, nodeID string, output any) error {
	if output != nil {
		fmt.Printf("更新节点上下文: %s, %+v\n", nodeID, output)
		ctx.SetVariable(nodeID+".output", output)
	}
	return nil
}

// createNodeResult 创建节点结果
func (e *WorkflowEngine) createNodeResult(node *core.NodeDefinition, input, output any, result *core.Result, startTime time.Time) *core.NodeResult {
	return &core.NodeResult{
		Input:    input,
		Output:   output,
		Route:    result.Route,
		NodeID:   node.ID,
		Duration: time.Since(startTime).Milliseconds(),
		Error:    "",
		Type:     node.Type,
	}
}

// handleNodeError 处理节点错误
func (e *WorkflowEngine) handleNodeError(ctx *core.ExecutionContext, node *core.NodeDefinition, err error, startTime time.Time) error {
	nodeResult := &core.NodeResult{
		NodeID:   node.ID,
		Duration: time.Since(startTime).Milliseconds(),
		Error:    err.Error(),
		Type:     node.Type,
	}
	e.updateWorkflowState(ctx, []*core.NodeResult{nodeResult})
	return err
}

// updateWorkflowState 更新工作流状态
func (e *WorkflowEngine) updateWorkflowState(ctx *core.ExecutionContext, results []*core.NodeResult) error {

	for _, result := range results {
		// fmt.Printf("更新工作流状态: %s, %+v\n", result.NodeID, result)
		ctx.SetNodeResult(result.NodeID, result)
	}

	// 计算整体进度
	completed := len(ctx.State.Result)
	total := int(ctx.TotalNodes)
	ctx.State.Progress = float64(completed) / float64(total)

	return nil
}

// GetNodeResult 获取指定工作流节点的执行结果
func (e *WorkflowEngine) GetNodeResult(workflowID, serialID, nodeID string) (*core.NodeResult, bool) {
	e.mu.RLock()
	executor, ok := e.executorPool[workflowID]
	e.mu.RUnlock()

	if !ok {
		return nil, false
	}

	// 使用sync.Map获取执行上下文
	val, ok := executor.execContexts.Load(serialID)
	if !ok {
		return nil, false
	}

	execCtx, ok := val.(*core.ExecutionContext)
	if !ok {
		return nil, false
	}

	return execCtx.GetNodeResult(nodeID)
}

// buildExecutionPlan 构建执行计划
func (e *WorkflowEngine) buildExecutionPlan(ctx context.Context, def *core.WorkflowDef) (*ExecutionPlan, error) {
	// 构建依赖图
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	// 初始化入度
	for nodeID, def := range def.Nodes {
		inDegree[nodeID] = 0
		graph[nodeID] = []string{}
		// 迭代组件初始化
		if def.Type == "iteration" {
			fmt.Printf("迭代组件初始化: %s\n", def.ID)
			err := e.RegisterWorkflow(ctx, def.SubWorkflow)
			if err != nil {
				return nil, fmt.Errorf("迭代组件初始化失败: %v", err)
			}
		}
	}

	// 构建图结构
	for _, conn := range def.Connections {
		graph[conn.From] = append(graph[conn.From], conn.To)
		inDegree[conn.To]++
	}

	// 拓扑排序构建执行阶段
	var phases []ExecutionPhase
	for len(inDegree) > 0 {
		var phaseNodes []*WorkflowNode
		for nodeID, degree := range inDegree {
			if degree == 0 {
				node := def.Nodes[nodeID]
				phaseNodes = append(phaseNodes, &WorkflowNode{
					NodeDefinition: node,
				})
				delete(inDegree, nodeID)
			}
		}

		if len(phaseNodes) == 0 {
			return nil, fmt.Errorf("工作流中存在循环依赖")
		}

		phases = append(phases, ExecutionPhase{Nodes: phaseNodes})

		// 更新入度
		for _, node := range phaseNodes {
			for _, next := range graph[node.ID] {
				inDegree[next]--
			}
		}
	}
	return &ExecutionPlan{Phases: phases}, nil
}

// buildConditionRouter 构建条件路由
func (e *WorkflowEngine) buildConditionRouter(def *core.WorkflowDef) (map[string][]string, error) {
	router := make(map[string][]string)
	for _, conn := range def.Connections {
		router[conn.To] = append(router[conn.To], conn.From+"_"+conn.Condition)
	}
	return router, nil
}

// 启动清理协程
func (e *WorkflowEngine) startCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(e.config.CleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				e.cleanupExecContexts()
				e.adjustPoolSize()
			case workflowID := <-e.cleanup:
				e.mu.Lock()
				delete(e.executorPool, workflowID)
				e.mu.Unlock()
			case <-e.shutdownCh:
				return
			}
		}
	}()
}

// 启动指标收集器
func (e *WorkflowEngine) startMetricsCollector() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// 收集指标
				e.collectMetrics()
			case <-e.shutdownCh:
				return
			}
		}
	}()
}

// 收集指标
func (e *WorkflowEngine) collectMetrics() {
	// 这里可以实现指标收集和导出逻辑
}

// 调整池大小
func (e *WorkflowEngine) adjustPoolSize() {
	// 获取当前运行的任务数
	running := e.pool.Running()
	cap := e.pool.Cap()

	// 如果使用率超过80%，且未达到最大值，则增加容量
	if float64(running)/float64(cap) > 0.8 && cap < e.config.MaxPoolSize {
		newCap := int(float64(cap) * 1.5)
		if newCap > e.config.MaxPoolSize {
			newCap = e.config.MaxPoolSize
		}
		e.pool.Tune(newCap)
		e.poolSize = newCap
	}

	// 如果使用率低于30%，且池大小超过初始值，则减少容量
	if running > 0 && float64(running)/float64(cap) < 0.3 && cap > e.config.InitialPoolSize {
		newCap := int(float64(cap) * 0.7)
		if newCap < e.config.InitialPoolSize {
			newCap = e.config.InitialPoolSize
		}
		e.pool.Tune(newCap)
		e.poolSize = newCap
	}
}

// cleanupExecContexts 清理执行上下文
func (e *WorkflowEngine) cleanupExecContexts() {
	now := time.Now()

	e.mu.RLock()
	executors := make([]*Executor, 0, len(e.executorPool))
	for _, exec := range e.executorPool {
		executors = append(executors, exec)
	}
	e.mu.RUnlock()

	for _, executor := range executors {
		// 清理每个执行器的上下文
		executor.execContexts.Range(func(key, value any) bool {
			id := key.(string)
			ctx, ok := value.(*core.ExecutionContext)
			if !ok {
				executor.execContexts.Delete(key)
				return true
			}

			// 检查是否已完成或过期
			if ctx.State.Status == core.StatusCompleted ||
				ctx.State.Status == core.StatusFailed ||
				now.After(ctx.Expiration) {
				executor.execContexts.Delete(id)
				core.ReleaseExecutionContext(ctx) // 释放上下文
			}

			return true
		})
	}
}

// Cleanup 释放资源
func (e *WorkflowEngine) Cleanup() {
	close(e.shutdownCh)
	e.pool.Release()

	// 关闭所有执行器
	e.mu.Lock()
	for id, executor := range e.executorPool {
		close(executor.shutdownCh)
		delete(e.executorPool, id)
		fmt.Printf("关闭执行器: %s\n", executor.definition.ID)
	}
	e.mu.Unlock()
}

// GetMetrics 获取指定工作流的指标
func (e *WorkflowEngine) GetMetrics(workflowID string) (*core.WorkflowMetrics, bool) {
	if !e.config.EnableMetrics {
		return nil, false
	}

	e.metricsCollector.mu.RLock()
	defer e.metricsCollector.mu.RUnlock()

	metrics, ok := e.metricsCollector.metrics[workflowID]
	return metrics, ok
}

// GetPoolStats 获取池统计信息
func (e *WorkflowEngine) GetPoolStats() map[string]any {
	return map[string]any{
		"capacity":    e.pool.Cap(),
		"running":     e.pool.Running(),
		"free":        e.pool.Free(),
		"waiting":     e.pool.Waiting(),
		"poolSize":    e.poolSize,
		"maxPoolSize": e.config.MaxPoolSize,
		"initialSize": e.config.InitialPoolSize,
	}
}

// ListWorkflows 列出所有工作流
func (e *WorkflowEngine) ListWorkflows() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	workflowIDs := make([]string, 0, len(e.executorPool))
	for id := range e.executorPool {
		workflowIDs = append(workflowIDs, id)
	}

	return workflowIDs
}

// GetWorkflowStatus 获取工作流状态
func (e *WorkflowEngine) GetWorkflowStatus(workflowID string) (core.WorkflowStatus, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	executor, ok := e.executorPool[workflowID]
	if !ok {
		return "", false
	}

	return executor.status, true
}

// PauseWorkflow 暂停工作流执行
func (e *WorkflowEngine) PauseWorkflow(ctx context.Context, workflowID string, serialID string) error {
	// 获取工作流执行器
	e.mu.RLock()
	executor, ok := e.executorPool[workflowID]
	e.mu.RUnlock()

	if !ok {
		return fmt.Errorf("工作流未找到: %s", workflowID)
	}

	// 检查工作流状态
	if executor.status == core.WorkflowStatusShutdown {
		return fmt.Errorf("工作流正在关闭: %s", workflowID)
	}

	// 使用sync.Map获取执行上下文
	val, ok := executor.execContexts.Load(serialID)
	if !ok {
		return fmt.Errorf("工作流执行实例未找到: %s, %s", workflowID, serialID)
	}

	execCtx, ok := val.(*core.ExecutionContext)
	if !ok {
		return fmt.Errorf("工作流执行上下文类型错误: %s, %s", workflowID, serialID)
	}

	// 检查执行状态
	if execCtx.State.Status != core.StatusRunning {
		return fmt.Errorf("工作流执行状态不是运行中: %s, %s, 当前状态: %s",
			workflowID, serialID, execCtx.State.Status)
	}

	// 更新执行状态为暂停
	execCtx.State.Status = core.StatusPaused

	// 获取并调用取消函数
	cancelVal, ok := execCtx.GetVariable("cancel")
	if !ok {
		return fmt.Errorf("无法获取取消函数: %s, %s", workflowID, serialID)
	}

	cancel, ok := cancelVal.(context.CancelFunc)
	if !ok {
		return fmt.Errorf("取消函数类型错误: %s, %s", workflowID, serialID)
	}

	// 调用取消函数
	cancel()

	// 更新指标
	if e.config.EnableMetrics {
		atomic.AddInt64(&executor.metrics.PausedExecutions, 1)
	}

	return nil
}
