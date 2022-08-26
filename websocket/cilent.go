package websocket

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/riverchu/pkg/log"
)

// ConnectWebsocket connect websocket
func ConnectWebsocket(ctx context.Context, url string, header http.Header) (*websocket.Conn, *http.Response, error) {
	return websocket.DefaultDialer.DialContext(ctx, url, header)
}

// Read read from websocket
func Read(c *websocket.Conn) <-chan []byte {
	msg := make(chan []byte, 64)
	go func() {
		defer close(msg)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Info("read: %s", err)
				return
			}
			msg <- message
		}
	}()
	return msg
}

// Write write to websocket
func Write(c *websocket.Conn, msg []byte) error {
	return c.WriteMessage(websocket.TextMessage, msg)
}

func Close(c *websocket.Conn) error {
	// Cleanly close the connection by sending a close message and then
	// waiting (with timeout) for the server to close the connection.
	err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		return fmt.Errorf("write close: %s", err)
	}
	<-time.After(time.Second)
	return nil
}
