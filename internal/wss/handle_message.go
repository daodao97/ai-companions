package wss

import (
	"companions/internal/character"
	"companions/internal/conf"
	"companions/internal/pkg/xagent"
	"companions/internal/pkg/xllm"
	"companions/internal/pkg/xtts"
	"context"
	"encoding/json"

	"github.com/daodao97/xgo/xlog"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type AudioMessageResp struct {
	Type      string `json:"type"`
	Format    string `json:"format"`
	Data      string `json:"data"`
	MessageId string `json:"message_id"`
}

func handleTextMessage(conn *websocket.Conn, c *gin.Context, data []byte) {
	var textMsg TextMessage
	if err := json.Unmarshal(data, &textMsg); err != nil {
		xlog.ErrorC(context.Background(), "文本消息解析错误", xlog.Err(err))
		return
	}

	xlog.InfoC(context.Background(), "收到文本消息", xlog.Any("data", textMsg.Data))

	uid := c.Query("uid")

	llmConf := conf.Get().GetLLM("default")
	llm := xllm.New(llmConf)
	ttsConf := conf.Get().GetTTS("default")
	tts := xtts.New(ttsConf)

	// 使用新的工作流处理消息
	ctx := context.Background()
	agent := character.NewAgent(llm, tts, character.Ani)
	messageStream, err := agent.Execute(ctx, xagent.NewInput(map[string]any{
		"user_message": textMsg.Data,
		"user_id":      uid,
	}))
	if err != nil {
		xlog.ErrorC(ctx, "启动工作流失败", xlog.Err(err))
		if err := conn.WriteJSON(map[string]any{
			"type":    "error",
			"message": "工作流启动失败",
		}); err != nil {
			xlog.ErrorC(ctx, "发送错误响应失败", xlog.Err(err))
		}
		return
	}

	// 处理消息流并发送到WebSocket
	go func() {
		defer func() {
			if r := recover(); r != nil {
				xlog.ErrorC(ctx, "处理消息流时发生panic", xlog.Any("panic", r))
			}
		}()

		m := xagent.NewMessageProcessor()
		for msg := range messageStream {
			if !m.ShouldSend(msg) {
				continue
			}

			switch v := msg.(type) {
			case *character.AudioMessage:
				response := AudioMessageResp{
					Type:      "audio",
					Format:    v.AudioChunk.Format,
					Data:      v.AudioChunk.Data,
					MessageId: v.MessageID,
				}
				if err := sendMessage(conn, response); err != nil {
					xlog.ErrorC(ctx, "发送音频响应失败", xlog.Err(err))
					return
				}
			case *xagent.BaseMessage:
				response := map[string]any{
					"type": "text",
					"data": v.GetContent(),
				}
				if err := sendMessage(conn, response); err != nil {
					xlog.ErrorC(ctx, "发送文本响应失败", xlog.Err(err))
					return
				}
			case *character.RomanceMessage:
				response := map[string]any{
					"type": "romance",
					"data": v.Romance,
				}
				if err := sendMessage(conn, response); err != nil {
					xlog.ErrorC(ctx, "发送浪漫响应失败", xlog.Err(err))
					return
				}
			case *character.ActionMessage:
				response := map[string]any{
					"type": "action",
					"data": map[string]any{
						"action": v.Action,
						"args":   v.Args,
					},
				}
				if err := sendMessage(conn, response); err != nil {
					xlog.ErrorC(ctx, "发送动作响应失败", xlog.Err(err))
					return
				}
			case *character.ErrorMessage:
				response := map[string]any{
					"type":    "error",
					"message": v.Error,
				}
				if err := sendMessage(conn, response); err != nil {
					xlog.ErrorC(ctx, "发送错误响应失败", xlog.Err(err))
					return
				}
			}
		}

		xlog.InfoC(ctx, "消息流处理完成")
	}()
}

func sendMessage(conn *websocket.Conn, message any) error {
	return conn.WriteJSON(message)
}
