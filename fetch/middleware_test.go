package fetch

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// TestLogger implements RequestLogger for testing
type TestLogger struct {
	Requests  []string
	Responses []string
}

func (l *TestLogger) LogRequest(ctx context.Context, method, url string, headers http.Header, body []byte) {
	l.Requests = append(l.Requests, method+" "+url)
}

func (l *TestLogger) LogResponse(ctx context.Context, method, url string, statusCode int, headers http.Header, body []byte, duration time.Duration, err error) {
	response := method + " " + url + " - " + string(rune(statusCode))
	if err != nil {
		response += " - " + err.Error()
	}
	l.Responses = append(l.Responses, response)
}

// TestMetrics implements RequestMetrics for testing
type TestMetrics struct {
	Records []string
}

func (m *TestMetrics) RecordRequest(method, url string, statusCode int, duration time.Duration, err error) {
	record := method + " " + url + " - " + string(rune(statusCode)) + " - " + duration.String()
	if err != nil {
		record += " - " + err.Error()
	}
	m.Records = append(m.Records, record)
}

func TestWithLogging(t *testing.T) {
	logger := &TestLogger{}

	// Create a simple function to wrap
	fn := func() (int, []byte, http.Header, error) {
		return 200, []byte("test response"), http.Header{"Content-Type": []string{"application/json"}}, nil
	}

	middleware := WithLogging(logger)
	statusCode, content, _, err := middleware(fn)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if statusCode != 200 {
		t.Errorf("expected status 200, got %d", statusCode)
	}
	if string(content) != "test response" {
		t.Errorf("expected content 'test response', got %s", string(content))
	}

	// Verify logging occurred
	if len(logger.Responses) == 0 {
		t.Error("expected response logging, got none")
	}
}

func TestWithMetrics(t *testing.T) {
	metrics := &TestMetrics{}

	fn := func() (int, []byte, http.Header, error) {
		return 200, []byte("test response"), nil, nil
	}

	middleware := WithMetrics(metrics)
	statusCode, _, _, err := middleware(fn)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if statusCode != 200 {
		t.Errorf("expected status 200, got %d", statusCode)
	}

	// Verify metrics were recorded
	if len(metrics.Records) == 0 {
		t.Error("expected metrics recording, got none")
	}
}

func TestWithRequestTimeout(t *testing.T) {
	// Test with a function that takes longer than timeout
	fn := func() (int, []byte, http.Header, error) {
		time.Sleep(200 * time.Millisecond)
		return 200, []byte("delayed response"), nil, nil
	}

	middleware := WithRequestTimeout(100 * time.Millisecond)
	_, _, _, err := middleware(fn)

	if err == nil {
		t.Error("expected timeout error, got nil")
	}
	if err != ErrRequestTimeout {
		t.Errorf("expected ErrRequestTimeout, got %v", err)
	}

	// Test with a function that completes before timeout
	fn2 := func() (int, []byte, http.Header, error) {
		return 200, []byte("quick response"), nil, nil
	}

	middleware2 := WithRequestTimeout(1 * time.Second)
	statusCode, content, _, err := middleware2(fn2)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if statusCode != 200 {
		t.Errorf("expected status 200, got %d", statusCode)
	}
	if string(content) != "quick response" {
		t.Errorf("expected content 'quick response', got %s", string(content))
	}
}

func TestChainMiddleware(t *testing.T) {
	logger := &TestLogger{}
	metrics := &TestMetrics{}

	// Create a chain of middleware
	chain := ChainMiddleware(
		WithLogging(logger),
		WithMetrics(metrics),
	)

	fn := func() (int, []byte, http.Header, error) {
		return 200, []byte("chained response"), nil, nil
	}

	statusCode, content, _, err := chain(fn)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if statusCode != 200 {
		t.Errorf("expected status 200, got %d", statusCode)
	}
	if string(content) != "chained response" {
		t.Errorf("expected content 'chained response', got %s", string(content))
	}

	// Verify both middleware executed
	if len(logger.Responses) == 0 {
		t.Error("expected logging in chain, got none")
	}
	if len(metrics.Records) == 0 {
		t.Error("expected metrics in chain, got none")
	}
}

func TestSimpleLogger(t *testing.T) {
	logger := &SimpleLogger{}

	// Test LogRequest (should not panic)
	logger.LogRequest(context.Background(), "GET", "http://test.com", http.Header{}, []byte("body"))

	// Test LogResponse with error
	logger.LogResponse(context.Background(), "GET", "http://test.com", 500, http.Header{}, []byte("error"), 100*time.Millisecond, ErrRequestTimeout)

	// Test LogResponse without error
	logger.LogResponse(context.Background(), "GET", "http://test.com", 200, http.Header{}, []byte("success"), 50*time.Millisecond, nil)
}

func TestSimpleMetrics(t *testing.T) {
	metrics := &SimpleMetrics{}

	// Test RecordRequest with error
	metrics.RecordRequest("GET", "http://test.com", 500, 100*time.Millisecond, ErrRequestTimeout)

	// Test RecordRequest without error
	metrics.RecordRequest("GET", "http://test.com", 200, 50*time.Millisecond, nil)
}

func TestMiddlewareOrder(t *testing.T) {
	var executionOrder []string

	middleware1 := func(next func() (int, []byte, http.Header, error)) (int, []byte, http.Header, error) {
		executionOrder = append(executionOrder, "middleware1-before")
		result, content, headers, err := next()
		executionOrder = append(executionOrder, "middleware1-after")
		return result, content, headers, err
	}

	middleware2 := func(next func() (int, []byte, http.Header, error)) (int, []byte, http.Header, error) {
		executionOrder = append(executionOrder, "middleware2-before")
		result, content, headers, err := next()
		executionOrder = append(executionOrder, "middleware2-after")
		return result, content, headers, err
	}

	fn := func() (int, []byte, http.Header, error) {
		executionOrder = append(executionOrder, "function")
		return 200, []byte("test"), nil, nil
	}

	chain := ChainMiddleware(middleware1, middleware2)
	_, _, _, err := chain(fn)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify execution order: middleware1 -> middleware2 -> function -> middleware2 -> middleware1
	expectedOrder := []string{
		"middleware1-before",
		"middleware2-before",
		"function",
		"middleware2-after",
		"middleware1-after",
	}

	if len(executionOrder) != len(expectedOrder) {
		t.Errorf("expected %d execution steps, got %d: %v", len(expectedOrder), len(executionOrder), executionOrder)
	} else {
		for i, expected := range expectedOrder {
			if executionOrder[i] != expected {
				t.Errorf("step %d: expected %s, got %s", i, expected, executionOrder[i])
			}
		}
	}
}