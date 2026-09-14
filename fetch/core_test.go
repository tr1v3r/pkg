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

// slowUntilDisconnect sleeps ~2s unless the client goes away first, so a
// deadline-firing client keeps srv.Close() from blocking for the full sleep.
func slowUntilDisconnect(w http.ResponseWriter, r *http.Request) {
	select {
	case <-time.After(2 * time.Second):
	case <-r.Context().Done():
	}
	_, _ = io.WriteString(w, "finally")
}

// TestDoRequestWithContextOptionRegression reproduces BUG-AUDIT-REPORT H1:
// DoRequestWithContext used to append WithContext(ctx) AFTER the caller's opts,
// so the full context replacement silently dropped every context-carried option
// in opts — WithTimeout never fired (a 150ms budget against a 2s endpoint ran
// the full 2s with err=nil), WithMaxResponseBodySize and WithMiddleware were
// ignored, and the timeout cancel func was never called. The context option
// must be applied first, mirroring CtxGet/CtxPost/CtxPatch.
func TestDoRequestWithContextOptionRegression(t *testing.T) {
	middlewareRan := false

	cases := []struct {
		name       string
		ctx        context.Context
		handler    http.HandlerFunc
		opts       []RequestOption
		maxElapsed time.Duration // 0 means no upper bound
		verify     func(t *testing.T, statusCode int, content []byte, err error)
	}{
		{
			name:    "WithTimeout in opts still enforces the deadline",
			ctx:     context.Background(),
			handler: slowUntilDisconnect,
			opts:    []RequestOption{WithTimeout(150 * time.Millisecond)},
			// Contract: abort within 2x the 150ms budget, never the endpoint's full 2s.
			maxElapsed: 2 * 150 * time.Millisecond,
			verify: func(t *testing.T, _ int, _ []byte, err error) {
				if err == nil {
					t.Fatal("expected timeout error, got nil: WithTimeout in opts was dropped")
				}
			},
		},
		{
			name: "WithMaxResponseBodySize in opts still truncates the body",
			ctx:  context.Background(),
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, strings.Repeat("x", 4096))
			},
			opts: []RequestOption{WithMaxResponseBodySize(64)},
			verify: func(t *testing.T, _ int, content []byte, _ error) {
				if len(content) != 64 {
					t.Errorf("body len = %d, want 64: WithMaxResponseBodySize in opts was dropped", len(content))
				}
			},
		},
		{
			name: "WithMiddleware in opts still runs",
			ctx:  context.Background(),
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, "ok")
			},
			opts: []RequestOption{WithMiddleware(func(next func() (int, []byte, http.Header, error)) (int, []byte, http.Header, error) {
				middlewareRan = true
				return next()
			})},
			verify: func(t *testing.T, _ int, _ []byte, _ error) {
				if !middlewareRan {
					t.Error("middleware did not run: WithMiddleware in opts was dropped")
				}
			},
		},
		{
			name:       "ctx itself still propagates",
			ctx:        func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }(),
			handler:    slowUntilDisconnect,
			maxElapsed: time.Second,
			verify: func(t *testing.T, _ int, _ []byte, err error) {
				if err == nil {
					t.Error("expected error for already-cancelled ctx, got nil")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(tc.handler)
			defer srv.Close()

			start := time.Now()
			statusCode, content, err := DoRequestWithContext(tc.ctx, http.MethodGet, srv.URL, tc.opts, nil)
			elapsed := time.Since(start)
			tc.verify(t, statusCode, content, err)

			if tc.maxElapsed > 0 && elapsed > tc.maxElapsed {
				t.Errorf("request took %v, want it aborted well before %v", elapsed, tc.maxElapsed)
			}
		})
	}
}

// TestDoRequestWithContextOptsNotAliased guards the fix's secondary contract:
// DoRequestWithContext must not write into the caller's opts backing array
// (the old append(opts, WithContext(ctx)) could clobber a spare-capacity slot).
func TestDoRequestWithContextOptsNotAliased(t *testing.T) {
	srv := newTestBackend(t)

	opts := make([]RequestOption, 1, 4)
	opts[0] = WithUserAgent("regression-agent")

	_, _, err := DoRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/json", opts, nil)
	if err != nil {
		t.Fatalf("DoRequestWithContext fail: %v", err)
	}
	if len(opts) != 1 {
		t.Fatalf("caller opts len = %d, want 1", len(opts))
	}
	for i, opt := range opts[:cap(opts)] {
		if i >= len(opts) && opt != nil {
			t.Errorf("caller opts backing array was mutated at index %d", i)
		}
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
