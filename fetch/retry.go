package fetch

import (
	"io"
	"math"
	"math/rand"
	"net/http"
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

// WithRetry performs a request with retry logic
func WithRetry(config RetryConfig, fn func() (int, []byte, http.Header, error)) (int, []byte, http.Header, error) {
	var lastErr error
	var lastStatusCode int
	var lastContent []byte
	var lastHeaders http.Header

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		statusCode, content, headers, err := fn()
		lastStatusCode = statusCode
		lastContent = content
		lastHeaders = headers

		if err == nil {
			// Check if we should retry based on status code
			shouldRetry := false
			for _, retryCode := range config.RetryOnStatus {
				if statusCode == retryCode {
					shouldRetry = true
					break
				}
			}

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

		// Calculate delay with exponential backoff and jitter
		delay := calculateBackoff(config, attempt)
		time.Sleep(delay)
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
	backoff = backoff * jitter

	// Cap at max delay
	if backoff > float64(config.MaxDelay) {
		backoff = float64(config.MaxDelay)
	}

	return time.Duration(backoff)
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

// DoRequestWithRetry performs an HTTP request with retry logic
func DoRequestWithRetry(method string, url string, opts []RequestOption, body io.Reader, retryOpts ...RetryOption) (int, []byte, http.Header, error) {
	config := NewRetryConfig(retryOpts...)

	fn := func() (int, []byte, http.Header, error) {
		return DoRequestWithOptions(method, url, opts, body)
	}

	return WithRetry(config, fn)
}