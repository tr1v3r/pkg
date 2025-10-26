package fetch

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// RequestLogger defines the interface for logging HTTP requests
type RequestLogger interface {
	LogRequest(ctx context.Context, method, url string, headers http.Header, body []byte)
	LogResponse(ctx context.Context, method, url string, statusCode int, headers http.Header, body []byte, duration time.Duration, err error)
}

// RequestMetrics defines the interface for collecting request metrics
type RequestMetrics interface {
	RecordRequest(method, url string, statusCode int, duration time.Duration, err error)
}

// Middleware wraps HTTP request execution with additional functionality
type Middleware func(next func() (int, []byte, http.Header, error)) (int, []byte, http.Header, error)

// WithLogging adds request/response logging
func WithLogging(logger RequestLogger) Middleware {
	return func(next func() (int, []byte, http.Header, error)) (int, []byte, http.Header, error) {
		// Extract context from the request if available
		var ctx context.Context
		// In a real implementation, you would extract context from the request
		// For now, we'll use background context
		ctx = context.Background()

		// Log request (in a real implementation, you'd capture request details)
		// logger.LogRequest(ctx, method, url, headers, body)

		start := time.Now()
		statusCode, content, headers, err := next()
		duration := time.Since(start)

		// Log response
		logger.LogResponse(ctx, "", "", statusCode, headers, content, duration, err)

		return statusCode, content, headers, err
	}
}

// WithMetrics adds request metrics collection
func WithMetrics(metrics RequestMetrics) Middleware {
	return func(next func() (int, []byte, http.Header, error)) (int, []byte, http.Header, error) {
		start := time.Now()
		statusCode, content, headers, err := next()
		duration := time.Since(start)

		metrics.RecordRequest("", "", statusCode, duration, err)

		return statusCode, content, headers, err
	}
}

// WithRequestTimeout adds a timeout to the request
func WithRequestTimeout(timeout time.Duration) Middleware {
	return func(next func() (int, []byte, http.Header, error)) (int, []byte, http.Header, error) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		// In a real implementation, you would pass the context to the request
		// For now, we'll just simulate the timeout behavior
		done := make(chan struct{})
		var statusCode int
		var content []byte
		var headers http.Header
		var requestErr error

		go func() {
			statusCode, content, headers, requestErr = next()
			close(done)
		}()

		select {
		case <-done:
			return statusCode, content, headers, requestErr
		case <-ctx.Done():
			return 0, nil, nil, ErrRequestTimeout
		}
	}
}

// ChainMiddleware chains multiple middleware together
func ChainMiddleware(middlewares ...Middleware) Middleware {
	return func(next func() (int, []byte, http.Header, error)) (int, []byte, http.Header, error) {
		chain := next
		for i := len(middlewares) - 1; i >= 0; i-- {
			middleware := middlewares[i]
			chain = func(inner func() (int, []byte, http.Header, error)) func() (int, []byte, http.Header, error) {
				return func() (int, []byte, http.Header, error) {
					return middleware(inner)
				}
			}(chain)
		}
		return chain()
	}
}

// SimpleLogger is a basic implementation of RequestLogger
type SimpleLogger struct{}

// LogRequest logs HTTP request details
func (l *SimpleLogger) LogRequest(ctx context.Context, method, url string, headers http.Header, body []byte) {
	fmt.Printf("[REQUEST] %s %s\n", method, url)
}

// LogResponse logs HTTP response details
func (l *SimpleLogger) LogResponse(ctx context.Context, method, url string, statusCode int, headers http.Header, body []byte, duration time.Duration, err error) {
	if err != nil {
		fmt.Printf("[RESPONSE] %s %s - %d - %v - %s\n", method, url, statusCode, err, duration)
	} else {
		fmt.Printf("[RESPONSE] %s %s - %d - %s\n", method, url, statusCode, duration)
	}
}

// SimpleMetrics is a basic implementation of RequestMetrics
type SimpleMetrics struct{}

// RecordRequest records request metrics
func (m *SimpleMetrics) RecordRequest(method, url string, statusCode int, duration time.Duration, err error) {
	// In a real implementation, you would send metrics to a monitoring system
	fmt.Printf("[METRICS] %s %s - %d - %s\n", method, url, statusCode, duration)
}