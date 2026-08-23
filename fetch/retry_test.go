package fetch

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRetryConfig(t *testing.T) {
	config := NewRetryConfig(
		WithMaxAttempts(5),
		WithBaseDelay(100*time.Millisecond),
		WithMaxDelay(5*time.Second),
		WithJitter(0.1),
	)

	if config.MaxAttempts != 5 {
		t.Errorf("expected MaxAttempts=5, got %d", config.MaxAttempts)
	}
	if config.BaseDelay != 100*time.Millisecond {
		t.Errorf("expected BaseDelay=100ms, got %v", config.BaseDelay)
	}
	if config.MaxDelay != 5*time.Second {
		t.Errorf("expected MaxDelay=5s, got %v", config.MaxDelay)
	}
	if config.Jitter != 0.1 {
		t.Errorf("expected Jitter=0.1, got %v", config.Jitter)
	}
}

func TestRetryableStatusCode(t *testing.T) {
	testCases := []struct {
		statusCode int
		expected   bool
	}{
		{http.StatusOK, false},
		{http.StatusBadRequest, false},
		{http.StatusRequestTimeout, true},
		{http.StatusTooManyRequests, true},
		{http.StatusInternalServerError, true},
		{http.StatusBadGateway, true},
		{http.StatusServiceUnavailable, true},
		{http.StatusGatewayTimeout, true},
	}

	for _, tc := range testCases {
		result := IsRetryableStatusCode(tc.statusCode)
		if result != tc.expected {
			t.Errorf("IsRetryableStatusCode(%d) = %v, expected %v", tc.statusCode, result, tc.expected)
		}
	}
}

func TestRetrySuccess(t *testing.T) {
	attempts := 0
	fn := func() (int, []byte, http.Header, error) {
		attempts++
		if attempts == 1 {
			return http.StatusInternalServerError, []byte("error"), nil, nil
		}
		return http.StatusOK, []byte("success"), nil, nil
	}

	config := RetryConfig{
		MaxAttempts:   3,
		BaseDelay:     10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		Jitter:        0.0,
		RetryOnStatus: []int{http.StatusInternalServerError},
	}

	statusCode, content, _, err := WithRetry(context.Background(), config, fn)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", statusCode)
	}
	if string(content) != "success" {
		t.Errorf("expected content 'success', got %s", string(content))
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestRetryMaxAttempts(t *testing.T) {
	attempts := 0
	fn := func() (int, []byte, http.Header, error) {
		attempts++
		return http.StatusInternalServerError, []byte("error"), nil, nil
	}

	config := RetryConfig{
		MaxAttempts:   3,
		BaseDelay:     10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		Jitter:        0.0,
		RetryOnStatus: []int{http.StatusInternalServerError},
	}

	statusCode, content, _, err := WithRetry(context.Background(), config, fn)

	if err == nil {
		t.Error("expected error, got nil")
	}
	if _, ok := err.(*RetryableError); !ok {
		t.Errorf("expected RetryableError, got %T", err)
	}
	if statusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", statusCode)
	}
	if string(content) != "error" {
		t.Errorf("expected content 'error', got %s", string(content))
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryNetworkError(t *testing.T) {
	attempts := 0
	fn := func() (int, []byte, http.Header, error) {
		attempts++
		return 0, nil, nil, errors.New("network error")
	}

	config := RetryConfig{
		MaxAttempts:   3,
		BaseDelay:     10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		Jitter:        0.0,
		RetryOnStatus: []int{},
	}

	statusCode, content, headers, err := WithRetry(context.Background(), config, fn)
	_ = statusCode
	_ = content
	_ = headers

	if err == nil {
		t.Error("expected error, got nil")
	}
	if _, ok := err.(*RetryableError); !ok {
		t.Errorf("expected RetryableError, got %T", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestParseRetryAfter(t *testing.T) {
	cases := []struct {
		name  string
		value string
		check func(time.Duration) bool
		desc  string
	}{
		{"empty", "", func(d time.Duration) bool { return d == 0 }, "0"},
		{"seconds", "3", func(d time.Duration) bool { return d == 3*time.Second }, "3s"},
		{"http-date future", time.Now().Add(2 * time.Hour).UTC().Format(http.TimeFormat), func(d time.Duration) bool { return d > time.Hour }, ">1h"},
		{"http-date past", time.Now().Add(-2 * time.Hour).UTC().Format(http.TimeFormat), func(d time.Duration) bool { return d == 0 }, "0"},
		{"garbage", "not-a-date", func(d time.Duration) bool { return d == 0 }, "0"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseRetryAfter(tc.value)
			if !tc.check(got) {
				t.Errorf("parseRetryAfter(%q) = %v, want %s", tc.value, got, tc.desc)
			}
		})
	}
}

func TestRetryAfterHeaderRespected(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = io.WriteString(w, "recovered")
	}))
	defer srv.Close()

	status, content, _, err := DoRequestWithRetry("GET", srv.URL, nil, nil,
		WithMaxAttempts(3),
		WithBaseDelay(time.Millisecond),
		WithMaxDelay(5*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("retry with Retry-After fail: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want %d", status, http.StatusOK)
	}
	if string(content) != "recovered" {
		t.Errorf("content = %q, want %q", content, "recovered")
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}

func TestCalculateBackoff(t *testing.T) {
	config := NewRetryConfig(
		WithBaseDelay(10*time.Millisecond),
		WithMaxDelay(50*time.Millisecond),
		WithJitter(0.1),
	)

	if d := calculateBackoff(config, 0); d != 10*time.Millisecond {
		t.Errorf("attempt 0 backoff = %v, want base delay 10ms", d)
	}

	// exponential growth must be capped at MaxDelay
	for attempt := 1; attempt < 10; attempt++ {
		if d := calculateBackoff(config, attempt); d > 50*time.Millisecond {
			t.Errorf("attempt %d backoff = %v, exceeds max 50ms", attempt, d)
		}
	}
}

func TestRetryableErrorUnwrap(t *testing.T) {
	inner := errors.New("boom")
	re := &RetryableError{Err: inner, Attempts: 3}

	if re.Error() == "" {
		t.Error("Error() should not be empty")
	}
	if !errors.Is(re, inner) {
		t.Error("errors.Is should unwrap to inner error")
	}
}
