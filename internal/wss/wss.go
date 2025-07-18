package wss

import (
	_ "embed"
	"encoding/json"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/daodao97/xgo/xapp"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// 连接统计
var (
	activeConnections int64
	totalConnections  int64
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域连接
	},
	HandshakeTimeout: 45 * time.Second,
}

// 消息类型定义
type Message struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

type TextMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

type AudioMessage struct {
	Type   string `json:"type"`
	Format string `json:"format"`
	Size   int    `json:"size"`
	Data   string `json:"data"`
}

type PingMessage struct {
	Type string `json:"type"`
}

type PongMessage struct {
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp"`
}

func SetupRouter(e *gin.Engine) {
	e.GET("/ws", func(c *gin.Context) {
		// 连接统计
		connID := atomic.AddInt64(&totalConnections, 1)
		atomic.AddInt64(&activeConnections, 1)

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("WebSocket升级失败: %v", err)
			atomic.AddInt64(&activeConnections, -1)
			return
		}
		defer func() {
			conn.Close()
			atomic.AddInt64(&activeConnections, -1)
			active := atomic.LoadInt64(&activeConnections)
			log.Printf("WebSocket连接已断开 [ID:%d] (当前活跃连接: %d)", connID, active)
		}()

		// 记录连接信息
		clientIP := c.ClientIP()
		userAgent := c.GetHeader("User-Agent")
		active := atomic.LoadInt64(&activeConnections)
		log.Printf("新的WebSocket连接建立 [ID:%d] IP:%s UA:%s (当前活跃连接: %d)", connID, clientIP, userAgent, active)

		// 设置连接参数
		conn.SetReadLimit(512 * 1024) // 512KB
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		// 处理消息循环
		for {
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				// 检查是否是正常关闭
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
					log.Printf("WebSocket异常断开 [ID:%d]: %v", connID, err)
				} else {
					// 正常关闭，使用INFO级别
					if closeErr, ok := err.(*websocket.CloseError); ok {
						switch closeErr.Code {
						case websocket.CloseNormalClosure:
							log.Printf("WebSocket正常关闭 [ID:%d]", connID)
						case websocket.CloseGoingAway:
							log.Printf("WebSocket客户端离开 [ID:%d] (页面刷新/关闭)", connID)
						case websocket.CloseNoStatusReceived:
							log.Printf("WebSocket连接异常中断 [ID:%d]", connID)
						default:
							log.Printf("WebSocket关闭 [ID:%d]: code=%d, text=%s", connID, closeErr.Code, closeErr.Text)
						}
					} else {
						log.Printf("WebSocket连接结束 [ID:%d]: %v", connID, err)
					}
				}
				break
			}

			// 重置读取超时
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))

			if messageType == websocket.TextMessage {
				handleMessage(conn, c, p)
			}
		}
	})

	e.GET("/", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.HTML(http.StatusOK, "index.html", gin.H{
			"IsDev": xapp.IsDev(),
		})
	})

	e.GET("/ws_test", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.File("./assets/wss_test.html")
	})
	// 连接统计接口
	e.GET("/ws_stats", func(c *gin.Context) {
		active := atomic.LoadInt64(&activeConnections)
		total := atomic.LoadInt64(&totalConnections)
		c.JSON(http.StatusOK, gin.H{
			"active_connections": active,
			"total_connections":  total,
		})
	})
}

func handleMessage(conn *websocket.Conn, c *gin.Context, data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("JSON解析错误: %v", err)
		return
	}

	log.Printf("收到消息类型: %s", msg.Type)

	switch msg.Type {
	case "ping":
		handlePing(conn)
	case "text":
		handleTextMessage(conn, c, data)
	case "audio":
		handleAudioMessage(conn, c, data)
	default:
		log.Printf("未知消息类型: %s", msg.Type)
	}
}
