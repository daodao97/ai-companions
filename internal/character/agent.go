package character

import (
	"companions/internal/dao"
	"companions/internal/pkg/xagent"
	"companions/internal/pkg/xflow"
	"companions/internal/pkg/xllm"
	"companions/internal/pkg/xmem"
	"companions/internal/pkg/xtools"
	"companions/internal/pkg/xtts"
	"context"
	"fmt"
	"strings"

	"github.com/daodao97/xgo/xlog"
)

type AgentOption func(a *Agent)

type Agent struct {
	xagent.BaseAgent
	character *Character
	tts       xtts.TTS
	flow      *xflow.Flow[AICompanionState]
	xtools    *xtools.Tools
	tools     []xllm.Tool
}

func WithXTools(tools ...xtools.ToolInterface) AgentOption {
	var _xllmTools []xllm.Tool
	for _, tool := range tools {
		schema := tool.GetSchema()
		schema.Name = gentFunctionName("xtools", schema.Name)
		_xllmTools = append(_xllmTools, schema)
	}

	return func(a *Agent) {
		a.xtools = xtools.NewTools(tools...)
		a.tools = append(a.tools, _xllmTools...)
	}
}

func gentFunctionName(serverName string, toolName string) string {
	return fmt.Sprintf("%s___%s", serverName, toolName)
}

func SplitFunctionName(functionName string) (string, string) {
	tokens := strings.Split(functionName, "___")
	return tokens[0], tokens[1]
}

func NewAgent(llm xllm.LLM, tts xtts.TTS, character *Character) *Agent {
	return &Agent{
		BaseAgent: *xagent.NewBaseAgent(character.Name, character.Instructions, llm, []xllm.Parameter{
			{
				Name:        "character",
				Type:        "string",
				Description: "角色",
			},
		}),
		character: character,
		tts:       tts,
	}
}

func (a *Agent) Execute(ctx context.Context, input xagent.Input) (chan xagent.Message, error) {
	mem := xmem.NewMysqlMemory(dao.ConversationModel, dao.MessageModel)
	state := &AICompanionState{
		UserMessage:   input["user_message"].(string),
		UserID:        input["user_id"].(string),
		Memory:        mem,
		MessageStream: make(chan xagent.Message, 100),
		LLM:           a.GetLLM(),
		TTS:           a.tts,
	}

	// 构建消息历史
	var allMsg = []xllm.Message{}
	// 获取历史记忆
	if state.UserID != "" && len(state.History) == 0 {
		history, _ := state.Memory.GetMemory(state.UserID)
		allMsg = append(allMsg, history...)
		state.History = allMsg
	}

	flow := a.GetFlow(state)

	go func() {
		defer func() {
			xlog.Debug("关闭 MessageStream")
			close(state.MessageStream)
			xlog.Debug("MessageStream 已关闭")
		}()

		err := flow.Execute(ctx)
		if err != nil {
			xlog.Error("工作流执行失败", xlog.Err(err))
			// 发送错误消息
			errorMsg := NewErrorMessage(err.Error())
			select {
			case state.MessageStream <- errorMsg:
			default:
			}
		}
	}()

	return state.MessageStream, nil
}

func (a *Agent) GetFlow(state *AICompanionState) *xflow.Flow[AICompanionState] {
	if a.flow == nil {
		flow := xflow.NewFlow(state)
		// tips: 1, 2, 3 是并行执行的
		// 0. start
		// 1. llm chat response -> tts generate
		// 2. romance_meter_change
		// 3. action
		// 4. end

		// todo:  add 工具调用 搜索, 播放音乐 等 节点

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

		a.flow = flow
	}

	return a.flow
}
