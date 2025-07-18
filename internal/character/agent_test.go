package character

import (
	"companions/internal/conf"
	"companions/internal/dao"
	"companions/internal/pkg/xagent"
	"companions/internal/pkg/xflow"
	"companions/internal/pkg/xllm"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/daodao97/xgo/xapp"
	"github.com/daodao97/xgo/xdb"
)

func init() {
	xapp.SetConfDir("../../")
	if err := conf.InitConf(); err != nil {
		panic(err)
	}
	conf.Get().Database[0].DSN = "../../companion.db"
	if err := xdb.Inits(conf.Get().Database); err != nil {
		panic(err)
	}
	dao.Init()
}

func TestJoinEndNode(t *testing.T) {
	// 初始化配置
	if err := conf.InitConf(); err != nil {
		t.Fatalf("配置初始化失败: %v", err)
	}

	// 创建测试状态
	state := &AICompanionState{
		UserMessage:  "测试消息",
		RomanceMeter: 50,
	}

	// 创建工作流
	flow := xflow.NewFlow(state)

	// 添加节点
	startNode := xflow.NewStartNode("start")
	llmChatAndTTSNode := NewLLMChatAndTTSNode()
	romanceNode := NewRomanceMeterChangeNode()
	actionNode := NewActionNode()

	// 创建并行节点
	parallelNode := xflow.NewParallelNode[AICompanionState]("parallel_tasks",
		llmChatAndTTSNode,
		romanceNode,
		actionNode,
	)

	// 创建汇聚并结束节点
	joinEndNode := xflow.NewJoinEndNode[AICompanionState]("join_and_end", xflow.JoinAll)

	flow.AddNode(
		startNode,
		parallelNode,
		joinEndNode, // 既是汇聚又是结束
	)

	// 添加边
	flow.AddEdge(startNode, parallelNode)
	flow.AddEdge(parallelNode, joinEndNode)

	// 执行工作流
	ctx := context.Background()
	startTime := time.Now()

	err := flow.Execute(ctx)
	if err != nil {
		t.Fatalf("工作流执行失败: %v", err)
	}

	duration := time.Since(startTime)

	// 验证执行时间
	if duration > 200*time.Millisecond {
		t.Errorf("并行执行时间过长: %v，期望小于200ms", duration)
	}

	// 验证最终状态
	finalState := flow.GetState()
	if finalState.LLMResponse == "" {
		t.Error("LLM响应为空")
	}
	if finalState.RomanceMeter != 55 {
		t.Errorf("浪漫度错误，期望55，实际%d", finalState.RomanceMeter)
	}
	if finalState.ActionTaken == "" {
		t.Error("执行动作为空")
	}

	// 验证执行记录
	trace := flow.GetExecutionTrace()
	if len(trace) != 3 { // start -> parallel -> join_end
		t.Errorf("执行记录数量错误，期望3条，实际%d", len(trace))
	}

	// 验证最后一个节点是结束节点
	lastRecord := trace[len(trace)-1]
	if lastRecord.NodeName != "join_and_end" {
		t.Errorf("最后一个节点错误，期望join_and_end，实际%s", lastRecord.NodeName)
	}

	t.Logf("JoinEndNode工作流执行成功，耗时: %v", duration)
	t.Logf("最终状态: %+v", finalState)
}

func TestAgent(t *testing.T) {
	fmt.Println(conf.Get())
	llmConf := conf.Get().GetLLM("default")
	llm := xllm.New(llmConf)
	agent := NewAgent(llm, Ani)
	resp, err := agent.Execute(
		context.Background(),
		xagent.NewInput(map[string]any{
			"user_message": "hello",
			"user_id":      "test",
		}),
	)
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}
	for msg := range resp {
		fmt.Println(xagent.NewMessageProcessor().ToJSON(msg))
	}
}
