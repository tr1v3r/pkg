package websocket_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	ws "github.com/tr1v3r/pkg/websocket"
)

const (
	serverScheme = "ws"
	serverAddr   = "localhost:" + serverPort
	serverPath   = "/ws"
	serverPort   = "7750"
)

var server = &url.URL{Scheme: serverScheme, Host: serverAddr, Path: serverPath}

func TestConnectWebsocket(t *testing.T) {
	c, _, err := ws.ConnectWebsocket(context.Background(), server.String(), nil)
	if err != nil {
		t.Errorf("connect server websocket fail: %s", err)
		return
	}
	if c == nil {
		t.Errorf("connect server websocket fail: got nil")
		return
	}
	defer ws.Close(c)
}

func TestConnectWebsocket_timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	c, _, err := ws.ConnectWebsocket(ctx, server.String(), nil)
	if err != nil {
		t.Errorf("connect server websocket fail: %s", err)
		return
	}
	if c == nil {
		t.Errorf("connect server websocket fail: got nil")
		return
	}
	defer ws.Close(c)
}

func TestConnectAndCommunicate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, _, err := ws.ConnectWebsocket(ctx, server.String(), nil)
	if err != nil {
		t.Errorf("connect server websocket fail: %s", err)
		return
	}
	if c == nil {
		t.Errorf("connect server websocket fail: got nil")
		return
	}
	defer ws.Close(c)

	// Limit message count to prevent infinite loop
	maxMessages := 5
	messageCount := 0

	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case ts := <-ticker.C:
				if messageCount >= maxMessages {
					return
				}
				err := ws.Write(c, []byte(ts.String()))
				if err != nil {
					t.Errorf("write msg fail: %s", err)
					return
				}
			}
		}
	}()

	// Read messages with timeout and limit
	for {
		select {
		case <-ctx.Done():
			t.Logf("Test completed by context timeout")
			return
		case msg, ok := <-ws.Read(c):
			if !ok {
				t.Logf("Read channel closed")
				return
			}
			t.Logf("recv: %s", string(msg))
			messageCount++
			if messageCount >= maxMessages {
				t.Logf("Received expected number of messages")
				return
			}
		}
	}
}
