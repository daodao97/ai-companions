package wss

import (
	"companions/internal/conf"
	"companions/internal/pkg/xstt"
	"context"
	"encoding/json"
	"log"

	"github.com/daodao97/xgo/xlog"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func handleAudioMessage(conn *websocket.Conn, c *gin.Context, data []byte) {
	var audioMsg AudioMessage
	if err := json.Unmarshal(data, &audioMsg); err != nil {
		log.Printf("音频消息解析错误: %v", err)
		return
	}

	log.Printf("收到音频消息: 格式=%s, 大小=%d字节", audioMsg.Format, audioMsg.Size)

	sttConf := conf.Get().GetSTT("default")

	if sttConf == nil {
		if err := conn.WriteJSON(map[string]any{
			"type":    "error",
			"message": "语音识别配置不存在",
		}); err != nil {
			xlog.ErrorC(context.Background(), "发送文本响应失败", xlog.Err(err))
		}
		return
	}

	stt := xstt.New(sttConf)

	res, err := stt.SpeechToText(context.Background(), xstt.SpeechToTextReq{
		Audio:  audioMsg.Data,
		Format: audioMsg.Format,
	})
	if err != nil {
		xlog.ErrorC(context.Background(), "语音识别失败: %v", err)
		return
	}
	xlog.InfoC(context.Background(), "语音识别结果: %v", res)

	toText := map[string]any{
		"type": "text",
		"data": res.Text,
	}

	jsonData, err := json.Marshal(toText)
	if err != nil {
		xlog.ErrorC(context.Background(), "JSON编码失败: %v", err)
		return
	}

	handleTextMessage(conn, c, jsonData)
}
