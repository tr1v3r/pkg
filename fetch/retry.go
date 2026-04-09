package fetch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"slices"
	"strconv"
	"time"
)

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxAttempts   int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	Jitter        float64 // 0.0 to 1.0
	RetryOnStatus []int   // Status codes to retry on
}

// DefaultRetryConfig provides sensible defaults
var DefaultRetryConfig = RetryConfig{
	MaxAttempts: 3,
	BaseDelay:   100 * time.Millisecond,
	MaxDelay:    10 * time.Second,
	Jitter:      0.2,
	RetryOnStatus: []int{
		http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
	},
}

// WithRetry performs a request with retry logic.
// The ctx parameter allows cancellation of retry waits between attempts.
// If config.MaxAttempts is 0 or less, fn is called exactly once with no retries.
func WithRetry(ctx context.Context, config RetryConfig, fn func() (int, []byte, http.Header, error)) (int, []byte, http.Header, error) {
	if config.MaxAttempts <= 0 {
		statusCode, content, headers, err := fn()
		if err != nil {
			return statusCode, content, headers, err
		}
		if slices.Contains(config.RetryOnStatus, statusCode) {
			return statusCode, content, headers, &RetryableError{
				Err:      &HTTPError{StatusCode: statusCode, Body: content},
				Attempts: 1,
			}
		}
		return statusCode, content, headers, nil
	}

	var lastErr error
	var lastStatusCode int
	var lastContent []byte
	var lastHeaders http.Header

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Check context before each attempt
		select {
		case <-ctx.Done():
			return lastStatusCode, lastContent, lastHeaders, ctx.Err()
		default:
		}

		statusCode, content, headers, err := fn()
		lastStatusCode = statusCode
		lastContent = content
		lastHeaders = headers

		if err == nil {
			// Check if we should retry based on status code
			shouldRetry := slices.Contains(config.RetryOnStatus, statusCode)

			if !shouldRetry {
				return statusCode, content, headers, nil
			}
			lastErr = &HTTPError{StatusCode: statusCode, Body: content}
		} else {
			lastErr = err
		}

		// Don't retry on last attempt
		if attempt == config.MaxAttempts-1 {
			break
		}

		// Calculate delay: prefer Retry-After header if present
		var delay time.Duration
		if retryAfter := parseRetryAfter(headers.Get("Retry-After")); retryAfter > 0 {
			delay = retryAfter
		} else {
			delay = calculateBackoff(config, attempt)
		}

		// Cancellable sleep
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return lastStatusCode, lastContent, lastHeaders, ctx.Err()
		}
	}

	return lastStatusCode, lastContent, lastHeaders, &RetryableError{
		Err:      lastErr,
		Attempts: config.MaxAttempts,
	}
}

// calculateBackoff calculates the delay with exponential backoff and jitter
func calculateBackoff(config RetryConfig, attempt int) time.Duration {
	if attempt == 0 {
		return config.BaseDelay
	}

	// Exponential backoff
	backoff := float64(config.BaseDelay) * math.Pow(2, float64(attempt))

	// Apply jitter
	jitter := 1.0 + config.Jitter*(rand.Float64()*2-1)
	backoff *= jitter

	// Cap at max delay
	if backoff > float64(config.MaxDelay) {
		backoff = float64(config.MaxDelay)
	}

	return time.Duration(backoff)
}

// parseRetryAfter parses the Retry-After header value.
// Supports both integer seconds and HTTP-date formats (RFC 7231).
// Returns 0 if the value cannot be parsed.
func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	// Try parsing as seconds (integer)
	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second
	}
	// Try parsing as HTTP-date (RFC 7231)
	if t, err := http.ParseTime(value); err == nil {
		remaining := time.Until(t)
		if remaining > 0 {
			return remaining
		}
	}
	return 0
}

// RetryOption is a functional option for configuring retry behavior
type RetryOption func(*RetryConfig)

// WithMaxAttempts sets the maximum number of retry attempts
func WithMaxAttempts(attempts int) RetryOption {
	return func(c *RetryConfig) {
		c.MaxAttempts = attempts
	}
}

// WithBaseDelay sets the base delay for exponential backoff
func WithBaseDelay(delay time.Duration) RetryOption {
	return func(c *RetryConfig) {
		c.BaseDelay = delay
	}
}

// WithMaxDelay sets the maximum delay for exponential backoff
func WithMaxDelay(delay time.Duration) RetryOption {
	return func(c *RetryConfig) {
		c.MaxDelay = delay
	}
}

// WithJitter sets the jitter factor (0.0 to 1.0)
func WithJitter(jitter float64) RetryOption {
	return func(c *RetryConfig) {
		c.Jitter = jitter
	}
}

// NewRetryConfig creates a new retry configuration with options
func NewRetryConfig(opts ...RetryOption) RetryConfig {
	config := DefaultRetryConfig
	for _, opt := range opts {
		opt(&config)
	}
	return config
}

// DoRequestWithRetryContext performs an HTTP request with retry logic and context support.
// The request body is buffered to allow safe retries.
func DoRequestWithRetryContext(ctx context.Context, method string, url string, opts []RequestOption, body io.Reader,
	retryOpts ...RetryOption) (int, []byte, http.Header, error) {
	config := NewRetryConfig(retryOpts...)

	// Buffer the body for retries
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return 0, nil, nil, fmt.Errorf("reading request body for retry: %w", err)
		}
	}

	fn := func() (int, []byte, http.Header, error) {
		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}
		return DoRequestWithOptions(method, url, opts, bodyReader)
	}

	return WithRetry(ctx, config, fn)
}

// DoRequestWithRetry performs an HTTP request with retry logic.
// The request body is buffered to allow safe retries.
func DoRequestWithRetry(method string, url string, opts []RequestOption, body io.Reader,
	retryOpts ...RetryOption) (int, []byte, http.Header, error) {
	return DoRequestWithRetryContext(context.Background(), method, url, opts, body, retryOpts...)
}
