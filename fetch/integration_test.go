package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// Test server that simulates various scenarios
type testServer struct {
	*httptest.Server
	requestCount int
	mu           sync.Mutex
}

func newTestServer() *testServer {
	ts := &testServer{}
	ts.Server = httptest.NewServer(http.HandlerFunc(ts.handler))
	return ts
}

func (ts *testServer) handler(w http.ResponseWriter, r *http.Request) {
	ts.mu.Lock()
	ts.requestCount++
	ts.mu.Unlock()

	path := r.URL.Path

	switch path {
	case "/ok":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w,`{"status": "ok"}`)

	case "/echo":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		body, _ := io.ReadAll(r.Body)
		_, _ = fmt.Fprintf(w,`{"echo": "%s"}`, string(body))

	case "/slow":
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w,`{"status": "slow"}`)

	case "/error/500":
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w,"Internal Server Error")

	case "/error/429":
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = fmt.Fprintf(w,"Rate Limited")

	case "/error/502":
		w.WriteHeader(http.StatusBadGateway)
		_, _ = fmt.Fprintf(w,"Bad Gateway")

	case "/headers":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w,`{"user-agent": "%s", "authorization": "%s"}`,
			r.Header.Get("User-Agent"),
			r.Header.Get("Authorization"))

	case "/count":
		ts.mu.Lock()
		count := ts.requestCount
		ts.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w,`{"count": %d}`, count)

	default:
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprintf(w,"Not Found")
	}
}

func (ts *testServer) reset() {
	ts.mu.Lock()
	ts.requestCount = 0
	ts.mu.Unlock()
}

func (ts *testServer) getRequestCount() int {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.requestCount
}

func TestIntegrationBasic(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	// Test basic GET
	data, err := Get(server.URL + "/ok")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	expected := `{"status": "ok"}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}
}

func TestIntegrationPost(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	body := strings.NewReader(`{"test": "data"}`)
	data, err := Post(server.URL+"/echo", body, WithContentTypeJSON())
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	expected := `{"echo": "{\"test\": \"data\"}"}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}
}

func TestIntegrationHeaders(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	opts := []RequestOption{
		WithUserAgent("TestAgent/1.0"),
		WithAuthToken("Bearer test-token"),
	}

	data, err := Get(server.URL+"/headers", opts...)
	if err != nil {
		t.Fatalf("Get with headers failed: %v", err)
	}

	expected := `{"user-agent": "TestAgent/1.0", "authorization": "Bearer test-token"}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}
}

func TestIntegrationRetry(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	server.reset()

	// Configure retry for 500 errors
	config := NewRetryConfig(
		WithMaxAttempts(3),
		WithBaseDelay(10*time.Millisecond),
	)

	fn := func() (int, []byte, http.Header, error) {
		return DoRequestWithOptions("GET", server.URL+"/error/500", nil, nil)
	}

	statusCode, _, _, err := WithRetry(config, fn)

	// Should get the 500 error after retries
	if statusCode != 500 {
		t.Errorf("expected status 500, got %d", statusCode)
	}
	if err == nil {
		t.Error("expected error after retries, got nil")
	}

	// Should have made 3 attempts
	count := server.getRequestCount()
	if count != 3 {
		t.Errorf("expected 3 requests, got %d", count)
	}
}

func TestIntegrationRetrySuccess(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	server.reset()

	// This endpoint returns 500 on first call, then 200
	callCount := 0
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintf(w,"Error")
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w,`{"status": "success"}`)
		}
	})

	config := NewRetryConfig(
		WithMaxAttempts(3),
		WithBaseDelay(10*time.Millisecond),
	)

	fn := func() (int, []byte, http.Header, error) {
		return DoRequestWithOptions("GET", server.URL, nil, nil)
	}

	statusCode, content, _, err := WithRetry(config, fn)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if statusCode != 200 {
		t.Errorf("expected status 200, got %d", statusCode)
	}
	if string(content) != `{"status": "success"}` {
		t.Errorf("unexpected content: %s", string(content))
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestIntegrationCircuitBreaker(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 1,
		OpenTimeout:      100 * time.Millisecond,
	})

	// Fail enough to open circuit
	for i := 0; i < 2; i++ {
		err := cb.Execute(func() error {
			_, err := Get(server.URL + "/error/500")
			return err
		})
		if err != nil {
			t.Logf("Execute error (expected): %v", err)
		}
	}

	// Circuit should be open
	if cb.State() != StateOpen {
		t.Errorf("expected circuit open, got %v", cb.State())
	}

	// Should get circuit breaker error
	err := cb.Execute(func() error {
		_, err := Get(server.URL + "/ok")
		return err
	})

	if err == nil {
		t.Error("expected circuit breaker error, got nil")
	}
	if _, ok := err.(*CircuitBreakerError); !ok {
		t.Errorf("expected CircuitBreakerError, got %T", err)
	}
}

func TestIntegrationContextCancellation(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := CtxGet(ctx, server.URL+"/slow")
	if err == nil {
		t.Error("expected error for cancelled context, got nil")
	}
}

func TestIntegrationTimeout(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := CtxGet(ctx, server.URL+"/slow")
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestIntegrationMultipleMethods(t *testing.T) {
	server := newTestServer()
	defer server.Close()

	// Test various HTTP methods
	tests := []struct {
		method string
		url    string
		body   io.Reader
	}{
		{"GET", "/ok", nil},
		{"POST", "/echo", strings.NewReader(`{"test": "data"}`)},
		{"PATCH", "/echo", strings.NewReader(`{"patch": "data"}`)},
	}

	for _, test := range tests {
		t.Run(test.method, func(t *testing.T) {
			statusCode, content, _, err := DoRequestWithOptions(test.method, server.URL+test.url, []RequestOption{WithContentTypeJSON()}, test.body)
			if err != nil {
				t.Fatalf("%s failed: %v", test.method, err)
			}
			if statusCode != 200 {
				t.Errorf("%s: expected status 200, got %d", test.method, statusCode)
			}
			if len(content) == 0 {
				t.Errorf("%s: returned empty content", test.method)
			}
		})
	}
}