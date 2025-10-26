package fetch

import (
	"errors"
	"fmt"
	"net/http"
)

// Custom error types for better error handling
type (
	// HTTPError represents an HTTP response error
	HTTPError struct {
		StatusCode int
		Body       []byte
		URL        string
	}

	// RetryableError indicates an error that can be retried
	RetryableError struct {
		Err      error
		Attempts int
	}

	// CircuitBreakerError indicates the circuit breaker is open
	CircuitBreakerError struct {
		Err error
	}
)

// Error implements the error interface
func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, string(e.Body))
}

// Error implements the error interface
func (e *RetryableError) Error() string {
	return fmt.Sprintf("retryable error (attempt %d): %v", e.Attempts, e.Err)
}

// Error implements the error interface
func (e *CircuitBreakerError) Error() string {
	return fmt.Sprintf("circuit breaker open: %v", e.Err)
}

// Unwrap returns the underlying error
func (e *RetryableError) Unwrap() error {
	return e.Err
}

// Unwrap returns the underlying error
func (e *CircuitBreakerError) Unwrap() error {
	return e.Err
}

// IsRetryableStatusCode checks if a status code is retryable
func IsRetryableStatusCode(statusCode int) bool {
	return statusCode == http.StatusRequestTimeout ||
		statusCode == http.StatusTooManyRequests ||
		statusCode == http.StatusInternalServerError ||
		statusCode == http.StatusBadGateway ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusGatewayTimeout
}

// IsClientError checks if the status code indicates a client error
func IsClientError(statusCode int) bool {
	return statusCode >= 400 && statusCode < 500
}

// IsServerError checks if the status code indicates a server error
func IsServerError(statusCode int) bool {
	return statusCode >= 500
}

// Common errors
var (
	ErrRateLimited     = errors.New("rate limited")
	ErrCircuitOpen     = errors.New("circuit breaker open")
	ErrMaxRetries      = errors.New("maximum retries exceeded")
	ErrInvalidResponse = errors.New("invalid response")
	ErrRequestTimeout  = errors.New("request timeout")
)