package fetch

import (
	"errors"
	"net/http"
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

	statusCode, content, _, err := WithRetry(config, fn)

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

	statusCode, content, _, err := WithRetry(config, fn)

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

	_, _, _, err := WithRetry(config, fn)

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