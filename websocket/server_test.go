package websocket_test

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/tr1v3r/pkg/log"

	ws "github.com/tr1v3r/pkg/websocket"
)

const port = "7749"

func TestServer(t *testing.T) {
	r := gin.Default()

	r.GET("/ws", ws.WSHanlder(handle))

	err := r.Run(":" + port)
	if err != nil {
		log.Error("gin server stopped: %s", err)
	}
}

func handle(_ *websocket.Conn, msg []byte) []byte {
	switch string(msg) {
	case "ping":
		return []byte("pong")
	default:
		return msg
	}
}
