package fetch

import (
	"errors"
	"net/http"
	"testing"
)

func TestHTTPError(t *testing.T) {
	err := &HTTPError{
		StatusCode: 404,
		Body:       []byte("Not Found"),
		URL:        "http://example.com/test",
	}

	expected := "HTTP 404: Not Found"
	if err.Error() != expected {
		t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
	}

	// Test with empty body
	err2 := &HTTPError{
		StatusCode: 500,
		Body:       []byte{},
		URL:        "http://example.com/test",
	}

	expected2 := "HTTP 500: "
	if err2.Error() != expected2 {
		t.Errorf("expected error message '%s', got '%s'", expected2, err2.Error())
	}
}

func TestRetryableError(t *testing.T) {
	originalErr := errors.New("original error")
	err := &RetryableError{
		Err:      originalErr,
		Attempts: 3,
	}

	expected := "retryable error (attempt 3): original error"
	if err.Error() != expected {
		t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
	}

	// Test Unwrap
	if err.Unwrap() != originalErr {
		t.Error("Unwrap() should return the original error")
	}
}

func TestCircuitBreakerError(t *testing.T) {
	originalErr := errors.New("circuit open")
	err := &CircuitBreakerError{
		Err: originalErr,
	}

	expected := "circuit breaker open: circuit open"
	if err.Error() != expected {
		t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
	}

	// Test Unwrap
	if err.Unwrap() != originalErr {
		t.Error("Unwrap() should return the original error")
	}
}

func TestIsRetryableStatusCode(t *testing.T) {
	testCases := []struct {
		statusCode int
		expected   bool
		desc       string
	}{
		{http.StatusOK, false, "200 OK"},
		{http.StatusCreated, false, "201 Created"},
		{http.StatusBadRequest, false, "400 Bad Request"},
		{http.StatusUnauthorized, false, "401 Unauthorized"},
		{http.StatusForbidden, false, "403 Forbidden"},
		{http.StatusNotFound, false, "404 Not Found"},
		{http.StatusRequestTimeout, true, "408 Request Timeout"},
		{http.StatusTooManyRequests, true, "429 Too Many Requests"},
		{http.StatusInternalServerError, true, "500 Internal Server Error"},
		{http.StatusBadGateway, true, "502 Bad Gateway"},
		{http.StatusServiceUnavailable, true, "503 Service Unavailable"},
		{http.StatusGatewayTimeout, true, "504 Gateway Timeout"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := IsRetryableStatusCode(tc.statusCode)
			if result != tc.expected {
				t.Errorf("IsRetryableStatusCode(%d) = %v, expected %v", tc.statusCode, result, tc.expected)
			}
		})
	}
}

func TestIsClientError(t *testing.T) {
	testCases := []struct {
		statusCode int
		expected   bool
		desc       string
	}{
		{199, false, "199 Informational"},
		{200, false, "200 OK"},
		{299, false, "299 Success"},
		{300, false, "300 Redirection"},
		{399, false, "399 Redirection"},
		{400, true, "400 Bad Request"},
		{401, true, "401 Unauthorized"},
		{404, true, "404 Not Found"},
		{499, true, "499 Client Error"},
		{500, false, "500 Server Error"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := IsClientError(tc.statusCode)
			if result != tc.expected {
				t.Errorf("IsClientError(%d) = %v, expected %v", tc.statusCode, result, tc.expected)
			}
		})
	}
}

func TestIsServerError(t *testing.T) {
	testCases := []struct {
		statusCode int
		expected   bool
		desc       string
	}{
		{199, false, "199 Informational"},
		{200, false, "200 OK"},
		{299, false, "299 Success"},
		{300, false, "300 Redirection"},
		{399, false, "399 Redirection"},
		{400, false, "400 Bad Request"},
		{499, false, "499 Client Error"},
		{500, true, "500 Server Error"},
		{501, true, "501 Not Implemented"},
		{502, true, "502 Bad Gateway"},
		{599, true, "599 Server Error"},
		{600, false, "600 Unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := IsServerError(tc.statusCode)
			if result != tc.expected {
				t.Errorf("IsServerError(%d) = %v, expected %v", tc.statusCode, result, tc.expected)
			}
		})
	}
}

func TestCommonErrors(t *testing.T) {
	// Test that common errors are defined
	if ErrRateLimited == nil {
		t.Error("ErrRateLimited should not be nil")
	}
	if ErrCircuitOpen == nil {
		t.Error("ErrCircuitOpen should not be nil")
	}
	if ErrMaxRetries == nil {
		t.Error("ErrMaxRetries should not be nil")
	}
	if ErrInvalidResponse == nil {
		t.Error("ErrInvalidResponse should not be nil")
	}
	if ErrRequestTimeout == nil {
		t.Error("ErrRequestTimeout should not be nil")
	}

	// Test error messages
	if ErrRateLimited.Error() != "rate limited" {
		t.Errorf("ErrRateLimited message incorrect: %s", ErrRateLimited.Error())
	}
	if ErrCircuitOpen.Error() != "circuit breaker open" {
		t.Errorf("ErrCircuitOpen message incorrect: %s", ErrCircuitOpen.Error())
	}
	if ErrMaxRetries.Error() != "maximum retries exceeded" {
		t.Errorf("ErrMaxRetries message incorrect: %s", ErrMaxRetries.Error())
	}
	if ErrInvalidResponse.Error() != "invalid response" {
		t.Errorf("ErrInvalidResponse message incorrect: %s", ErrInvalidResponse.Error())
	}
	if ErrRequestTimeout.Error() != "request timeout" {
		t.Errorf("ErrRequestTimeout message incorrect: %s", ErrRequestTimeout.Error())
	}
}

func TestErrorTypeAssertions(t *testing.T) {
	// Test that we can assert error types
	var err error

	httpErr := &HTTPError{StatusCode: 404, Body: []byte("test")}
	err = httpErr
	if _, ok := err.(*HTTPError); !ok {
		t.Error("HTTPError should be assertable as *HTTPError")
	}

	retryErr := &RetryableError{Err: errors.New("test"), Attempts: 1}
	err = retryErr
	if _, ok := err.(*RetryableError); !ok {
		t.Error("RetryableError should be assertable as *RetryableError")
	}

	cbErr := &CircuitBreakerError{Err: errors.New("test")}
	err = cbErr
	if _, ok := err.(*CircuitBreakerError); !ok {
		t.Error("CircuitBreakerError should be assertable as *CircuitBreakerError")
	}
}