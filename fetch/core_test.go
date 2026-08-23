package fetch

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestBackend starts a local HTTP backend equivalent to the endpoints this
// suite historically borrowed from httpbin.org, so tests are hermetic.
func newTestBackend(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"slideshow":{"title":"sample"}}`)
	})

	echo := func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"method":"`+r.Method+`","body":`+string(body)+`}`)
	}
	mux.HandleFunc("/post", echo)
	mux.HandleFunc("/patch", echo)

	mux.HandleFunc("/status/500", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	mux.HandleFunc("/headers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test-Header", "yes")
		_, _ = io.WriteString(w, "ok")
	})

	mux.HandleFunc("/delay/", func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second)
		_, _ = io.WriteString(w, "finally")
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestDefaultClient(t *testing.T) {
	client := DefaultClient()
	if client == nil {
		t.Error("DefaultClient() returned nil")
		return
	}
	if client.Timeout <= 0 {
		t.Errorf("expected positive timeout, got %v", client.Timeout)
	}
}

func TestSetDefaultClient(t *testing.T) {
	originalClient := DefaultClient()

	newClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	SetDefaultClient(newClient)

	currentClient := DefaultClient()
	if currentClient.Timeout != 30*time.Second {
		t.Errorf("expected timeout 30s after SetDefaultClient, got %v", currentClient.Timeout)
	}

	// Restore original client
	SetDefaultClient(originalClient)
}

func TestNewInsecureClient(t *testing.T) {
	client := NewInsecureClient()
	if client == nil {
		t.Error("NewInsecureClient() returned nil")
		return
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Error("client transport is not *http.Transport")
	}

	if transport.TLSClientConfig == nil {
		t.Error("TLSClientConfig is nil")
	}

	if !transport.TLSClientConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should be true for insecure client")
	}
}

func TestCtxGet(t *testing.T) {
	srv := newTestBackend(t)

	data, err := CtxGet(context.Background(), srv.URL+"/json")
	if err != nil {
		t.Fatalf("CtxGet fail: %v", err)
	}
	if !strings.Contains(string(data), "sample") {
		t.Errorf("CtxGet body = %s, want it to contain %q", data, "sample")
	}
}

func TestPost(t *testing.T) {
	srv := newTestBackend(t)
	body := strings.NewReader(`{"test": "data"}`)

	data, err := Post(srv.URL+"/post", body, WithContentTypeJSON())
	if err != nil {
		t.Fatalf("Post fail: %v", err)
	}
	if !strings.Contains(string(data), "test") {
		t.Errorf("Post body = %s, want it to contain echoed body", data)
	}
	if !strings.Contains(string(data), "POST") {
		t.Errorf("Post body = %s, want it to contain method POST", data)
	}
}

func TestCtxPost(t *testing.T) {
	srv := newTestBackend(t)
	ctx := context.Background()
	body := strings.NewReader(`{"test": "data"}`)

	data, err := CtxPost(ctx, srv.URL+"/post", body, WithContentTypeJSON())
	if err != nil {
		t.Fatalf("CtxPost fail: %v", err)
	}
	if len(data) == 0 {
		t.Error("CtxPost returned empty data")
	}
}

func TestPatch(t *testing.T) {
	srv := newTestBackend(t)
	body := strings.NewReader(`{"test": "data"}`)

	data, err := Patch(srv.URL+"/patch", body, WithContentTypeJSON())
	if err != nil {
		t.Fatalf("Patch fail: %v", err)
	}
	if !strings.Contains(string(data), "PATCH") {
		t.Errorf("Patch body = %s, want it to contain method PATCH", data)
	}
}

func TestCtxPatch(t *testing.T) {
	srv := newTestBackend(t)
	ctx := context.Background()
	body := strings.NewReader(`{"test": "data"}`)

	data, err := CtxPatch(ctx, srv.URL+"/patch", body, WithContentTypeJSON())
	if err != nil {
		t.Fatalf("CtxPatch fail: %v", err)
	}
	if len(data) == 0 {
		t.Error("CtxPatch returned empty data")
	}
}

func TestDoRequest(t *testing.T) {
	srv := newTestBackend(t)

	statusCode, content, err := DoRequest("GET", srv.URL+"/json", nil)
	if err != nil {
		t.Fatalf("DoRequest fail: %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", statusCode, http.StatusOK)
	}
	if len(content) == 0 {
		t.Error("DoRequest returned empty content")
	}
}

func TestDoRequestWithContext(t *testing.T) {
	srv := newTestBackend(t)
	ctx := context.Background()

	statusCode, content, err := DoRequestWithContext(ctx, "GET", srv.URL+"/json", nil, nil)
	if err != nil {
		t.Fatalf("DoRequestWithContext fail: %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", statusCode, http.StatusOK)
	}
	if len(content) == 0 {
		t.Error("DoRequestWithContext returned empty content")
	}
}

func TestDoRequestWithOptions(t *testing.T) {
	srv := newTestBackend(t)
	opts := []RequestOption{
		WithContentTypeJSON(),
		WithUserAgent("test-agent"),
	}

	statusCode, content, headers, err := DoRequestWithOptions("GET", srv.URL+"/headers", opts, nil)
	if err != nil {
		t.Fatalf("DoRequestWithOptions fail: %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", statusCode, http.StatusOK)
	}
	if len(content) == 0 {
		t.Error("DoRequestWithOptions returned empty content")
	}
	if headers == nil {
		t.Error("DoRequestWithOptions returned nil headers")
	}
	if headers.Get("X-Test-Header") != "yes" {
		t.Errorf("X-Test-Header = %q, want %q", headers.Get("X-Test-Header"), "yes")
	}
}

func TestDoRequestWithOptionsError(t *testing.T) {
	// Test with invalid URL
	statusCode, content, headers, err := DoRequestWithOptions("GET", "invalid-url", nil, nil)
	_ = statusCode
	_ = content
	_ = headers
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

func TestDoRequestWithOptionsWithBody(t *testing.T) {
	srv := newTestBackend(t)
	body := strings.NewReader(`{"test": "data"}`)

	statusCode, content, respHeaders, err := DoRequestWithOptions("POST", srv.URL+"/post", []RequestOption{WithContentTypeJSON()}, body)
	_ = respHeaders
	if err != nil {
		t.Fatalf("DoRequestWithOptions with body fail: %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", statusCode, http.StatusOK)
	}
	if !strings.Contains(string(content), "test") {
		t.Errorf("content = %s, want echoed body", content)
	}
}

func TestContextCancellation(t *testing.T) {
	srv := newTestBackend(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancel the context

	_, err := CtxGet(ctx, srv.URL+"/delay/1")
	if err == nil {
		t.Error("expected error for cancelled context, got nil")
	}
}

func TestTimeout(t *testing.T) {
	srv := newTestBackend(t)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Backend sleeps 2s; this must fail fast with the 200ms deadline.
	_, err := CtxGet(ctx, srv.URL+"/delay/2")
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}
