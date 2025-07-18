package wss

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

func handlePing(conn *websocket.Conn) {
	log.Printf("收到ping，发送pong")

	pong := PongMessage{
		Type:      "pong",
		Timestamp: time.Now().UnixMilli(),
	}

	if err := conn.WriteJSON(pong); err != nil {
		log.Printf("发送pong失败: %v", err)
	}
}
