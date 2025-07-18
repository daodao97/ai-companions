package xflow

import (
	"context"
	"fmt"

	"github.com/daodao97/xgo/xlog"
	"github.com/daodao97/xgo/xutil"
)

// 统一节点包装器接口
type NodeWrapper[T any] interface {
	Node
	ExecuteWithState(ctx context.Context, state *T) (*NodeResult[T], error)
	DecideWithState(ctx context.Context, state *T) (bool, error)
}

// 通用节点包装器 - 统一处理所有节点类型
type UniversalNodeWrapper[T any] struct {
	Node
	executeNode  ExecuteNode[T]
	decisionNode DecisionNode[T]
	parallelNode ParallelNode[T]
	joinNode     JoinNode[T]
}

func NewUniversalNodeWrapper[T any](node Node) *UniversalNodeWrapper[T] {
	wrapper := &UniversalNodeWrapper[T]{
		Node: node,
	}

	// 类型断言，检查是否为泛型节点
	if execNode, ok := node.(ExecuteNode[T]); ok {
		wrapper.executeNode = execNode
	}
	if decNode, ok := node.(DecisionNode[T]); ok {
		wrapper.decisionNode = decNode
	}
	if parallelNode, ok := node.(ParallelNode[T]); ok {
		wrapper.parallelNode = parallelNode
	}
	if joinNode, ok := node.(JoinNode[T]); ok {
		wrapper.joinNode = joinNode
	}
	// 如果是JoinEndNode，也设置joinNode
	if joinEndNode, ok := node.(*JoinEndNodeImpl[T]); ok {
		wrapper.joinNode = joinEndNode
	}

	return wrapper
}

func (w *UniversalNodeWrapper[T]) ExecuteWithState(ctx context.Context, state *T) (*NodeResult[T], error) {
	if w.executeNode != nil {
		return w.executeNode.Execute(ctx, state)
	}
	return nil, fmt.Errorf("节点 %s 不是执行节点", w.GetName())
}

func (w *UniversalNodeWrapper[T]) DecideWithState(ctx context.Context, state *T) (bool, error) {
	if w.decisionNode != nil {
		return w.decisionNode.Decide(ctx, state)
	}
	return false, fmt.Errorf("节点 %s 不是决策节点", w.GetName())
}

type Flow[T any] struct {
	nodeWrappers   map[string]NodeWrapper[T] // 统一存储所有节点包装器
	edges          map[string][]ConditionalEdge
	startNode      string
	state          *T
	executionTrace []ExecutionRecord // 记录执行流程
}

func NewFlow[T any](state *T) *Flow[T] {
	return &Flow[T]{
		nodeWrappers:   make(map[string]NodeWrapper[T]),
		edges:          make(map[string][]ConditionalEdge),
		state:          state,
		executionTrace: make([]ExecutionRecord, 0),
	}
}

// 添加节点方法 - 为所有节点创建统一包装器
func (f *Flow[T]) AddNode(node ...Node) *Flow[T] {
	for _, node := range node {
		// 为所有节点创建统一包装器
		f.nodeWrappers[node.GetName()] = NewUniversalNodeWrapper[T](node)

		if node.GetType() == NodeTypeStart {
			f.startNode = node.GetName()
		}

		// 如果是并行节点，自动添加其分支节点
		if node.GetType() == NodeTypeParallel {
			if parallelNode, ok := node.(ParallelNode[T]); ok {
				branches := parallelNode.GetParallelBranches()
				for _, branch := range branches {
					f.nodeWrappers[branch.GetName()] = NewUniversalNodeWrapper[T](branch)
				}
			}
		}
	}

	return f
}

// 添加无条件边
func (f *Flow[T]) AddEdge(from, to Node) *Flow[T] {
	return f.AddConditionalEdge(from, to, nil)
}

// 添加条件边
func (f *Flow[T]) AddConditionalEdge(from, to Node, condition *bool) *Flow[T] {
	if f.edges[from.GetName()] == nil {
		f.edges[from.GetName()] = make([]ConditionalEdge, 0)
	}

	edge := ConditionalEdge{
		From:      from.GetName(),
		To:        to.GetName(),
		Condition: condition,
	}

	f.edges[from.GetName()] = append(f.edges[from.GetName()], edge)
	return f
}

func (f *Flow[T]) IsExecuteNode(nodeName string) bool {
	wrapper, exists := f.nodeWrappers[nodeName]
	if !exists {
		return false
	}
	return wrapper.GetType() == NodeTypeExecute
}

func (f *Flow[T]) IsDecisionNode(nodeName string) bool {
	wrapper, exists := f.nodeWrappers[nodeName]
	if !exists {
		return false
	}
	return wrapper.GetType() == NodeTypeDecision
}

// 获取具体类型的节点 - 保持向后兼容
func (f *Flow[T]) GetExecuteNode(nodeName string) (ExecuteNode[T], bool) {
	wrapper, exists := f.nodeWrappers[nodeName]
	if !exists {
		return nil, false
	}
	if universalWrapper, ok := wrapper.(*UniversalNodeWrapper[T]); ok {
		if universalWrapper.executeNode != nil {
			return universalWrapper.executeNode, true
		}
	}
	return nil, false
}

func (f *Flow[T]) GetDecisionNode(nodeName string) (DecisionNode[T], bool) {
	wrapper, exists := f.nodeWrappers[nodeName]
	if !exists {
		return nil, false
	}
	if universalWrapper, ok := wrapper.(*UniversalNodeWrapper[T]); ok {
		if universalWrapper.decisionNode != nil {
			return universalWrapper.decisionNode, true
		}
	}
	return nil, false
}

// 根据决策结果获取下一个节点
func (f *Flow[T]) getNextNodeForDecision(nodeName string, decision bool) string {
	edges := f.edges[nodeName]

	// 首先查找匹配条件的边
	for _, edge := range edges {
		if edge.Condition != nil && *edge.Condition == decision {
			return edge.To
		}
	}

	// 如果没有找到匹配的条件边，查找无条件边
	for _, edge := range edges {
		if edge.Condition == nil {
			return edge.To
		}
	}

	return ""
}

func (f *Flow[T]) Execute(ctx context.Context) error {
	err := f.Validate()
	if err != nil {
		return fmt.Errorf("工作流验证失败: %w", err)
	}

	f.ClearExecutionTrace()

	ctx = context.WithValue(ctx, FlowStateKey, f.state)

	currentNodeName := f.startNode
	currentState := f.state // 使用state而不是input

	for currentNodeName != "" {
		wrapper, exists := f.nodeWrappers[currentNodeName]
		if !exists {
			break
		}

		xlog.Debug("当前节点", xlog.String("node", currentNodeName), xlog.String("type", wrapper.GetType().String()))

		record := ExecutionRecord{
			NodeName: currentNodeName,
			NodeType: wrapper.GetType().String(),
			Success:  true,
		}

		switch wrapper.GetType() {
		case NodeTypeExecute:
			xlog.DebugCtx(ctx, "执行节点", xlog.String("node", currentNodeName))
			result, err := xutil.Retry(ctx, func(ctx context.Context) (*NodeResult[T], error) {
				return wrapper.ExecuteWithState(ctx, currentState)
			})
			if err != nil {
				record.Success = false
				record.Error = err.Error()
				f.executionTrace = append(f.executionTrace, record)
				return fmt.Errorf("执行节点 %s 失败: %w", currentNodeName, err)
			}
			// 更新state而不是data
			if result.State != nil {
				currentState = result.State
			}
			f.executionTrace = append(f.executionTrace, record)

			currentNodeName = f.GetNextNode(currentNodeName)
			continue

		case NodeTypeDecision:
			xlog.DebugCtx(ctx, "决策节点", xlog.String("node", currentNodeName))
			decision, err := xutil.Retry(ctx, func(ctx context.Context) (bool, error) {
				return wrapper.DecideWithState(ctx, currentState)
			})
			if err != nil {
				record.Success = false
				record.Error = err.Error()
				f.executionTrace = append(f.executionTrace, record)
				return fmt.Errorf("决策节点 %s 失败: %w", currentNodeName, err)
			}

			record.Decision = &decision
			f.executionTrace = append(f.executionTrace, record)

			// 根据决策结果获取下一个节点
			currentNodeName = f.getNextNodeForDecision(currentNodeName, decision)
			continue

		case NodeTypeParallel:
			xlog.DebugCtx(ctx, "并行节点", xlog.String("node", currentNodeName))

			// 获取并行分支
			if universalWrapper, ok := wrapper.(*UniversalNodeWrapper[T]); ok && universalWrapper.parallelNode != nil {
				branches := universalWrapper.parallelNode.GetParallelBranches()

				// 执行并行分支
				parallelResult, err := f.executeParallelBranches(ctx, branches, currentState)
				if err != nil {
					record.Success = false
					record.Error = err.Error()
					f.executionTrace = append(f.executionTrace, record)
					return fmt.Errorf("并行节点 %s 失败: %w", currentNodeName, err)
				}

				// 记录并行执行结果
				record.Success = parallelResult.AllCompleted
				if parallelResult.Error != nil {
					record.Error = parallelResult.Error.Error()
				}
				f.executionTrace = append(f.executionTrace, record)

				// 并行节点完成后，移动到下一个节点
				currentNodeName = f.GetNextNode(currentNodeName)
				continue
			}

			// 如果不是并行节点，按普通节点处理
			f.executionTrace = append(f.executionTrace, record)
			currentNodeName = f.GetNextNode(currentNodeName)
			continue

		case NodeTypeJoin:
			xlog.DebugCtx(ctx, "汇聚节点", xlog.String("node", currentNodeName))
			// 汇聚节点主要用于同步，不需要特殊处理
			f.executionTrace = append(f.executionTrace, record)
			currentNodeName = f.GetNextNode(currentNodeName)
			continue

		case NodeTypeEnd:
			f.executionTrace = append(f.executionTrace, record)
			xlog.DebugCtx(ctx, "workflow end", xlog.String("detail", f.GetFlowDetail(ctx)))
			return nil

		default:
			// 其他节点类型，如起始节点，使用第一个无条件边
			f.executionTrace = append(f.executionTrace, record)
			currentNodeName = f.GetNextNode(currentNodeName)
			continue
		}
	}

	return nil
}

func (f *Flow[T]) GetFlowDetail(ctx context.Context) string {
	if len(f.executionTrace) == 0 {
		return "工作流尚未执行或执行记录为空"
	}

	var result string
	result += "工作流执行详情:\n"
	result += "===================\n"

	for i, record := range f.executionTrace {
		result += fmt.Sprintf("%d. 节点: %s (类型: %s)\n", i+1, record.NodeName, record.NodeType)

		if record.Success {
			result += "   状态: 成功\n"
		} else {
			result += "   状态: 失败\n"
			result += fmt.Sprintf("   错误: %s\n", record.Error)
		}

		if record.Decision != nil {
			if *record.Decision {
				result += "   决策结果: true\n"
			} else {
				result += "   决策结果: false\n"
			}
		}

		result += "\n"
	}

	result += "===================\n"
	result += fmt.Sprintf("总执行节点数: %d\n", len(f.executionTrace))

	// 统计各类型节点数量
	nodeTypeCount := make(map[string]int)
	successCount := 0
	for _, record := range f.executionTrace {
		nodeTypeCount[record.NodeType]++
		if record.Success {
			successCount++
		}
	}

	result += "节点类型统计:\n"
	for nodeType, count := range nodeTypeCount {
		result += fmt.Sprintf("  %s: %d个\n", nodeType, count)
	}

	result += fmt.Sprintf("执行成功率: %.2f%%\n", float64(successCount)/float64(len(f.executionTrace))*100)

	return result
}

func (f *Flow[T]) GetNextNode(nodeName string) string {
	edges := f.edges[nodeName]
	for _, edge := range edges {
		if edge.Condition == nil {
			return edge.To
		}
	}
	return ""
}

func (f *Flow[T]) Validate() error {
	if len(f.nodeWrappers) == 0 {
		return fmt.Errorf("工作流没有定义任何节点")
	}

	if f.startNode == "" {
		return fmt.Errorf("工作流缺少起始节点")
	}

	// 检查是否存在起始节点
	hasStartNode := false
	for _, wrapper := range f.nodeWrappers {
		if wrapper.GetType() == NodeTypeStart {
			hasStartNode = true
			break
		}
	}
	if !hasStartNode {
		return fmt.Errorf("工作流缺少起始节点")
	}

	// 检查是否存在结束节点
	hasEndNode := false
	for _, wrapper := range f.nodeWrappers {
		if wrapper.GetType() == NodeTypeEnd {
			hasEndNode = true
			break
		}
	}
	if !hasEndNode {
		return fmt.Errorf("工作流缺少结束节点")
	}

	// 检查每个节点是否都有出边(除了结束节点和并行分支中的节点)
	for name, wrapper := range f.nodeWrappers {
		if wrapper.GetType() == NodeTypeEnd {
			continue
		}

		// 检查是否为并行分支中的节点
		isParallelBranch := false
		for _, parallelWrapper := range f.nodeWrappers {
			if parallelWrapper.GetType() == NodeTypeParallel {
				if universalWrapper, ok := parallelWrapper.(*UniversalNodeWrapper[T]); ok && universalWrapper.parallelNode != nil {
					branches := universalWrapper.parallelNode.GetParallelBranches()
					for _, branch := range branches {
						if branch.GetName() == name {
							isParallelBranch = true
							break
						}
					}
				}
			}
		}

		if !isParallelBranch && len(f.edges[name]) == 0 {
			return fmt.Errorf("节点 %s 没有定义出边", name)
		}
	}

	// 检查决策节点的出边是否都有条件
	for name, wrapper := range f.nodeWrappers {
		if wrapper.GetType() == NodeTypeDecision {
			edges := f.edges[name]
			for _, edge := range edges {
				if edge.Condition == nil {
					return fmt.Errorf("决策节点 %s 的出边缺少条件定义", name)
				}
			}
		}
	}

	// 检查边的目标节点是否存在
	for from, edges := range f.edges {
		for _, edge := range edges {
			if _, exists := f.nodeWrappers[edge.To]; !exists {
				return fmt.Errorf("边 %s -> %s 的目标节点不存在", from, edge.To)
			}
		}
	}

	return nil
}

func (f *Flow[T]) GetState() *T {
	return f.state
}

// 清空执行记录
func (f *Flow[T]) ClearExecutionTrace() {
	f.executionTrace = make([]ExecutionRecord, 0)
}

// 获取执行记录
func (f *Flow[T]) GetExecutionTrace() []ExecutionRecord {
	return f.executionTrace
}

// 执行并行分支
func (f *Flow[T]) executeParallelBranches(ctx context.Context, branches []Node, state *T) (*ParallelResult[T], error) {
	if len(branches) == 0 {
		return &ParallelResult[T]{
			BranchResults: make(map[string]*NodeResult[T]),
			AllCompleted:  true,
		}, nil
	}

	// 创建结果通道
	resultChan := make(chan struct {
		branchName string
		result     *NodeResult[T]
		err        error
	}, len(branches))

	// 启动所有分支的并行执行
	for _, branch := range branches {
		go func(branchNode Node) {
			branchName := branchNode.GetName()
			// 获取分支节点包装器
			branchWrapper, exists := f.nodeWrappers[branchName]
			if !exists {
				resultChan <- struct {
					branchName string
					result     *NodeResult[T]
					err        error
				}{
					branchName: branchName,
					result:     nil,
					err:        fmt.Errorf("分支节点 %s 不存在", branchName),
				}
				return
			}

			// 执行分支节点
			var result *NodeResult[T]
			var err error

			switch branchWrapper.GetType() {
			case NodeTypeExecute:
				result, err = branchWrapper.ExecuteWithState(ctx, state)
			default:
				err = fmt.Errorf("不支持的分支节点类型: %s", branchWrapper.GetType().String())
			}

			resultChan <- struct {
				branchName string
				result     *NodeResult[T]
				err        error
			}{
				branchName: branchName,
				result:     result,
				err:        err,
			}
		}(branch)
	}

	// 收集所有分支的结果
	branchResults := make(map[string]*NodeResult[T])
	var lastError error

	for range branches {
		select {
		case result := <-resultChan:
			if result.err != nil {
				lastError = result.err
			}
			branchResults[result.branchName] = result.result
		case <-ctx.Done():
			return &ParallelResult[T]{
				BranchResults: branchResults,
				AllCompleted:  false,
				Error:         ctx.Err(),
			}, ctx.Err()
		}
	}

	return &ParallelResult[T]{
		BranchResults: branchResults,
		AllCompleted:  lastError == nil,
		Error:         lastError,
	}, nil
}
