package websocket_test

import (
	"context"
	"testing"
	"time"

	ws "github.com/tr1v3r/pkg/websocket"
)

func TestConnectWebsocket(t *testing.T) {
	srv := newTestServer(t)

	c, _, err := ws.ConnectWebsocket(context.Background(), wsDefaultURL(srv), nil)
	if err != nil {
		t.Fatalf("connect fail: %s", err)
	}
	if c == nil {
		t.Fatal("connect got nil conn")
	}
	defer c.Close()
}

func TestConnectWebsocketWithContextTimeout(t *testing.T) {
	srv := newTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	c, _, err := ws.ConnectWebsocket(ctx, wsURL(srv), nil)
	if err != nil {
		t.Fatalf("connect fail: %s", err)
	}
	defer c.Close()
}

func TestConnectWebsocketCancelledContext(t *testing.T) {
	srv := newTestServer(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, _, err := ws.ConnectWebsocket(ctx, wsURL(srv), nil); err == nil {
		t.Error("cancelled context should fail to connect")
	}
}

func TestConnectWebsocketRefused(t *testing.T) {
	// nothing listens on this port range
	if _, _, err := ws.ConnectWebsocket(context.Background(), "ws://127.0.0.1:1/ws", nil); err == nil {
		t.Error("unreachable server should fail to connect")
	}
}

func TestWriteReadRoundTrip(t *testing.T) {
	srv := newTestServer(t)

	c, _, err := ws.ConnectWebsocket(context.Background(), wsURL(srv), nil)
	if err != nil {
		t.Fatalf("connect fail: %s", err)
	}
	defer c.Close()

	if err := ws.Write(c, []byte("ping")); err != nil {
		t.Fatalf("write fail: %s", err)
	}

	msgCh := ws.Read(c)
	select {
	case msg, ok := <-msgCh:
		if !ok {
			t.Fatal("read channel closed before message")
		}
		if string(msg) != "pong" {
			t.Errorf("read = %q, want %q", msg, "pong")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for echo")
	}
}

func TestReadChannelClosesOnConnClose(t *testing.T) {
	srv := newTestServer(t)

	c, _, err := ws.ConnectWebsocket(context.Background(), wsURL(srv), nil)
	if err != nil {
		t.Fatalf("connect fail: %s", err)
	}

	msgCh := ws.Read(c)
	_ = c.Close()

	select {
	case _, ok := <-msgCh:
		if ok {
			t.Fatal("channel should be closed, got message")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("read channel not closed after conn close")
	}
}

func TestWriteAfterClose(t *testing.T) {
	srv := newTestServer(t)

	c, _, err := ws.ConnectWebsocket(context.Background(), wsURL(srv), nil)
	if err != nil {
		t.Fatalf("connect fail: %s", err)
	}
	_ = c.Close()

	if err := ws.Write(c, []byte("x")); err == nil {
		t.Error("write on closed conn should fail")
	}
}

func TestClose(t *testing.T) {
	srv := newTestServer(t)

	c, _, err := ws.ConnectWebsocket(context.Background(), wsURL(srv), nil)
	if err != nil {
		t.Fatalf("connect fail: %s", err)
	}

	// Close sends a close frame and waits up to 1s for the server.
	done := make(chan error, 1)
	go func() { done <- ws.Close(c) }()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Close fail: %s", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not return in time")
	}
	_ = c.Close()
}

func TestCloseAlreadyClosed(t *testing.T) {
	srv := newTestServer(t)

	c, _, err := ws.ConnectWebsocket(context.Background(), wsURL(srv), nil)
	if err != nil {
		t.Fatalf("connect fail: %s", err)
	}
	_ = c.Close()

	if err := ws.Close(c); err == nil {
		t.Error("Close on already-closed conn should fail")
	}
}
