package websocket

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/riverchu/pkg/log"
)

var upgrader = new(websocket.Upgrader)

func WSHanlder(handle func([]byte) []byte) gin.HandlerFunc {
	return WSHanlderWithUpgrader(upgrader, handle)
}

func WSHanlderWithUpgrader(upgrader *websocket.Upgrader, handle func([]byte) []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		for {
			mt, msg, err := ws.ReadMessage()
			if err != nil {
				log.Warn("read message fail: %s", err)
				break
			}

			err = ws.WriteMessage(mt, handle(msg))
			if err != nil {
				log.Warn("write message fail: %s", err)
				break
			}
		}
	}

}
