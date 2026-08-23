package fetch

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDoRequestWithOptions_MiddlewareChain(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "payload")
	}))
	defer srv.Close()

	wrapped := false
	mw := func(next func() (int, []byte, http.Header, error)) (int, []byte, http.Header, error) {
		status, body, hdrs, err := next()
		wrapped = true
		return status, append(body, "+mw"...), hdrs, err
	}

	status, body, _, err := DoRequestWithOptions("GET", srv.URL, []RequestOption{WithMiddleware(mw)}, nil)
	if err != nil {
		t.Fatalf("request fail: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("status = %d", status)
	}
	if !wrapped {
		t.Error("middleware was not invoked")
	}
	if string(body) != "payload+mw" {
		t.Errorf("body = %q, middleware result not applied", body)
	}
}

func TestDoRequestWithOptions_MaxBodySize(t *testing.T) {
	big := strings.Repeat("x", 10000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, big)
	}))
	defer srv.Close()

	// limited read
	_, body, _, err := DoRequestWithOptions("GET", srv.URL,
		[]RequestOption{WithMaxResponseBodySize(100)}, nil)
	if err != nil {
		t.Fatalf("request fail: %v", err)
	}
	if len(body) != 100 {
		t.Errorf("limited body = %d bytes, want 100", len(body))
	}

	// unlimited (-1)
	_, body, _, err = DoRequestWithOptions("GET", srv.URL,
		[]RequestOption{WithMaxResponseBodySize(-1)}, nil)
	if err != nil {
		t.Fatalf("unlimited request fail: %v", err)
	}
	if len(body) != len(big) {
		t.Errorf("unlimited body = %d bytes, want %d", len(body), len(big))
	}
}

func TestDoRequestWithOptions_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer srv.Close()

	if _, _, _, err := DoRequestWithOptions("GET", srv.URL,
		[]RequestOption{WithTimeout(150 * time.Millisecond)}, nil); err == nil {
		t.Error("request should time out")
	}
}

func TestDoRequestWithRetryContext_CancelDuringBackoff(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, _, _, err := DoRequestWithRetryContext(ctx, "GET", srv.URL, nil, nil,
		WithMaxAttempts(10),
		WithBaseDelay(300*time.Millisecond),
		WithMaxDelay(300*time.Millisecond),
	)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want context.DeadlineExceeded", err)
	}
	if elapsed > 2*time.Second {
		t.Errorf("elapsed = %v, cancellation during backoff should return promptly", elapsed)
	}
}

func TestWithRetry_SuccessAndFailure(t *testing.T) {
	cfg := NewRetryConfig(
		WithMaxAttempts(3),
		WithBaseDelay(time.Millisecond),
		WithMaxDelay(5*time.Millisecond),
	)

	// immediate success
	calls := 0
	_, body, _, err := WithRetry(context.Background(), cfg, func() (int, []byte, http.Header, error) {
		calls++
		return 200, []byte("ok"), nil, nil
	})
	if err != nil || string(body) != "ok" || calls != 1 {
		t.Errorf("success path: err=%v body=%q calls=%d", err, body, calls)
	}

	// persistent transport failure exhausts attempts
	calls = 0
	_, _, _, err = WithRetry(context.Background(), cfg, func() (int, []byte, http.Header, error) {
		calls++
		return -1, nil, nil, errors.New("boom")
	})
	var re *RetryableError
	if !errors.As(err, &re) {
		t.Errorf("err = %v (%T), want *RetryableError", err, err)
	}
	if calls != cfg.MaxAttempts {
		t.Errorf("calls = %d, want %d", calls, cfg.MaxAttempts)
	}
	if re.Attempts != cfg.MaxAttempts {
		t.Errorf("Attempts = %d, want %d", re.Attempts, cfg.MaxAttempts)
	}
}

func TestWithRetry_NonRetryableClientError(t *testing.T) {
	cfg := NewRetryConfig(WithMaxAttempts(5), WithBaseDelay(time.Millisecond), WithMaxDelay(5*time.Millisecond))

	calls := 0
	_, _, _, err := WithRetry(context.Background(), cfg, func() (int, []byte, http.Header, error) {
		calls++
		return http.StatusBadRequest, []byte("bad request"), nil, nil
	})
	// 4xx is returned as-is without retrying
	if err != nil {
		t.Errorf("4xx should not produce error, got %v", err)
	}
	if calls != 1 {
		t.Errorf("4xx should not retry, calls = %d", calls)
	}
}
