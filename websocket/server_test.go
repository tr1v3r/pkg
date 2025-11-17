package websocket_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	ws "github.com/tr1v3r/pkg/websocket"
)

const testServerPort = "7750"

func TestServer(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	r.GET("/ws", ws.WSHanlder(handle))

	// Create a server that can be shut down
	server := &http.Server{
		Addr:    ":" + testServerPort,
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Errorf("Server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Test basic server functionality
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:" + testServerPort + "/ws")
	if err != nil {
		t.Errorf("Failed to connect to server: %v", err)
		return
	}
	defer resp.Body.Close()

	// Verify it's a WebSocket upgrade request
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Errorf("Expected WebSocket upgrade, got status: %d", resp.StatusCode)
	}

	// Gracefully shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("Failed to shutdown server: %v", err)
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
