package xflow

import "context"

type flowContextKey struct{}

var FlowStateKey = flowContextKey{}

// 节点类型枚举
type NodeType int

const (
	NodeTypeStart NodeType = iota
	NodeTypeEnd
	NodeTypeExecute
	NodeTypeDecision
	NodeTypeParallel // 新增：并行节点类型
	NodeTypeJoin     // 新增：汇聚节点类型
)

func (nt NodeType) String() string {
	switch nt {
	case NodeTypeStart:
		return "Start"
	case NodeTypeEnd:
		return "End"
	case NodeTypeExecute:
		return "Execute"
	case NodeTypeDecision:
		return "Decision"
	case NodeTypeParallel:
		return "Parallel"
	case NodeTypeJoin:
		return "Join"
	default:
		return "Unknown"
	}
}

// 通用节点接口
type Node interface {
	GetName() string
	GetType() NodeType
}

// 基础节点结构
type BaseNode struct {
	Name string
	Type NodeType
}

func (b *BaseNode) GetName() string {
	return b.Name
}

func (b *BaseNode) GetType() NodeType {
	return b.Type
}

// 起始节点
type StartNode struct {
	BaseNode
}

func NewStartNode(name string) *StartNode {
	return &StartNode{
		BaseNode: BaseNode{
			Name: name,
			Type: NodeTypeStart,
		},
	}
}

// 结束节点
type EndNode struct {
	BaseNode
}

func NewEndNode() *EndNode {
	return &EndNode{
		BaseNode: BaseNode{
			Name: "end_node",
			Type: NodeTypeEnd,
		},
	}
}

// 执行节点接口 - 泛型化
type ExecuteNode[T any] interface {
	Node
	Execute(ctx context.Context, state *T) (*NodeResult[T], error)
}

// 节点结果 - 泛型化并增加State字段
type NodeResult[T any] struct {
	Success bool
	Data    any
	Error   error
	State   *T // 新增state传递
}

func ConditionalTrue() *bool {
	true := true
	return &true
}

func ConditionalFalse() *bool {
	false := false
	return &false
}

// 分支节点接口 - 泛型化
type DecisionNode[T any] interface {
	Node
	Decide(ctx context.Context, state *T) (bool, error)
}

// 条件边 - 支持决策分支
type ConditionalEdge struct {
	From      string
	To        string
	Condition *bool // nil表示无条件边，true/false表示条件边
}

// 执行记录结构
type ExecutionRecord struct {
	NodeName   string
	NodeType   string
	Success    bool
	Error      string
	Decision   *bool  // 仅用于决策节点
	ParallelID string // 新增：并行执行ID，用于标识并行分支
}

// 并行节点接口
type ParallelNode[T any] interface {
	Node
	GetParallelBranches() []Node // 获取并行分支的节点列表
}

// 汇聚节点接口
type JoinNode[T any] interface {
	Node
	GetJoinCondition() JoinCondition // 获取汇聚条件
}

// 汇聚条件类型
type JoinCondition int

const (
	JoinAll JoinCondition = iota // 等待所有分支完成
	JoinAny                      // 等待任意一个分支完成
	JoinN                        // 等待N个分支完成
)

// 并行执行结果
type ParallelResult[T any] struct {
	BranchResults map[string]*NodeResult[T] // 各分支的执行结果
	AllCompleted  bool                      // 是否所有分支都已完成
	Error         error                     // 并行执行错误
}

// 并行节点实现
type ParallelNodeImpl[T any] struct {
	BaseNode
	branches []Node
}

func NewParallelNode[T any](name string, branches ...Node) *ParallelNodeImpl[T] {
	return &ParallelNodeImpl[T]{
		BaseNode: BaseNode{
			Name: name,
			Type: NodeTypeParallel,
		},
		branches: branches,
	}
}

func (p *ParallelNodeImpl[T]) GetParallelBranches() []Node {
	return p.branches
}

// 汇聚节点实现
type JoinNodeImpl[T any] struct {
	BaseNode
	joinCondition JoinCondition
	requiredCount int // 用于JoinN类型
}

func NewJoinNode[T any](name string, condition JoinCondition, requiredCount ...int) *JoinNodeImpl[T] {
	count := 0
	if len(requiredCount) > 0 {
		count = requiredCount[0]
	}

	return &JoinNodeImpl[T]{
		BaseNode: BaseNode{
			Name: name,
			Type: NodeTypeJoin,
		},
		joinCondition: condition,
		requiredCount: count,
	}
}

func (j *JoinNodeImpl[T]) GetJoinCondition() JoinCondition {
	return j.joinCondition
}

func (j *JoinNodeImpl[T]) GetRequiredCount() int {
	return j.requiredCount
}

// 汇聚并结束节点实现 - 合并Join和End功能
type JoinEndNodeImpl[T any] struct {
	BaseNode
	joinCondition JoinCondition // 汇聚条件
	requiredCount int           // 需要完成的分支数量
}

func NewJoinEndNode[T any](name string, condition JoinCondition, requiredCount ...int) *JoinEndNodeImpl[T] {
	count := 0
	if len(requiredCount) > 0 {
		count = requiredCount[0]
	}

	return &JoinEndNodeImpl[T]{
		BaseNode: BaseNode{
			Name: name,
			Type: NodeTypeEnd, // 使用End类型，但具有Join功能
		},
		joinCondition: condition,
		requiredCount: count,
	}
}

func (j *JoinEndNodeImpl[T]) GetJoinCondition() JoinCondition {
	return j.joinCondition
}

func (j *JoinEndNodeImpl[T]) GetRequiredCount() int {
	return j.requiredCount
}
