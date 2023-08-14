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
	serverAddr   = "localhost:" + port
	serverPath   = "/ws"
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

	go func() {
		for ts := range time.Tick(time.Second) {
			err := ws.Write(c, []byte(ts.String()))
			if err != nil {
				t.Errorf("write msg fail: %s", err)
			}
		}
	}()

	for msg := range ws.Read(c) {
		t.Logf("recv: %s", string(msg))
	}
}
