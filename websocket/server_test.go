package websocket_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	ws "github.com/tr1v3r/pkg/websocket"
)

// newTestServer spins up an in-process echo server around WSHanlder.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/ws", ws.WSHanlder(echoHandle))
	r.GET("/ws-default", ws.WSHanlder(echoHandle))

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func echoHandle(_ *websocket.Conn, msg []byte) []byte {
	switch string(msg) {
	case "ping":
		return []byte("pong")
	default:
		return msg
	}
}

func wsURL(srv *httptest.Server) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
}

func wsDefaultURL(srv *httptest.Server) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws-default"
}

func TestServer_PlainGETIsRejected(t *testing.T) {
	srv := newTestServer(t)

	// A plain GET without upgrade headers cannot be upgraded; the handler
	// returns and gorilla replies 400.
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(srv.URL + "/ws")
	if err != nil {
		t.Fatalf("plain GET fail: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("plain GET status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestServer_Echo(t *testing.T) {
	srv := newTestServer(t)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(srv), nil)
	if err != nil {
		t.Fatalf("dial fail: %s", err)
	}
	defer conn.Close()

	for _, msg := range []string{"ping", "hello"} {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
			t.Fatalf("write %q fail: %s", msg, err)
		}
		_, got, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read after %q fail: %s", msg, err)
		}
		want := msg
		if msg == "ping" {
			want = "pong"
		}
		if string(got) != want {
			t.Errorf("echo = %q, want %q", got, want)
		}
	}
}

func TestServer_ClientDisconnect(t *testing.T) {
	srv := newTestServer(t)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(srv), nil)
	if err != nil {
		t.Fatalf("dial fail: %s", err)
	}

	// Abruptly close the client; the server read loop must break, not hang.
	_ = conn.Close()
	time.Sleep(100 * time.Millisecond)
}
