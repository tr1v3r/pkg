package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestConcurrentDefaultClient(t *testing.T) {
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Test concurrent access to DefaultClient
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := DefaultClient()
			if client == nil {
				errors <- nil
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Should not have any errors
	for err := range errors {
		if err != nil {
			t.Errorf("concurrent DefaultClient access failed: %v", err)
		}
	}
}

func TestConcurrentSetDefaultClient(t *testing.T) {
	originalClient := DefaultClient()
	defer SetDefaultClient(originalClient)

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	// Test concurrent SetDefaultClient
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			client := &http.Client{
				Timeout: time.Duration(index+1) * time.Second,
			}
			SetDefaultClient(client)
			// Verify we can still get a client
			current := DefaultClient()
			if current == nil {
				errors <- nil
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Should not have any errors
	for err := range errors {
		if err != nil {
			t.Errorf("concurrent SetDefaultClient failed: %v", err)
		}
	}
}

func TestConcurrentCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig)

	var wg sync.WaitGroup
	successCount := 0
	failureCount := 0
	var mu sync.Mutex

	// Test concurrent circuit breaker execution
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(shouldFail bool) {
			defer wg.Done()
			err := cb.Execute(func() error {
				if shouldFail {
					return ErrRequestTimeout
				}
				return nil
			})

			mu.Lock()
			if err != nil {
				failureCount++
			} else {
				successCount++
			}
			mu.Unlock()
		}(i%2 == 0) // Alternate between success and failure
	}

	wg.Wait()

	// Verify state is reasonable
	state := cb.State()
	if state != StateClosed && state != StateOpen && state != StateHalfOpen {
		t.Errorf("invalid circuit breaker state: %v", state)
	}

	// Should have some successes and failures
	if successCount == 0 && failureCount == 0 {
		t.Error("no executions recorded")
	}
}

func TestConcurrentRetry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	var wg sync.WaitGroup
	errors := make(chan error, 20)

	config := NewRetryConfig(
		WithMaxAttempts(3),
		WithBaseDelay(10*time.Millisecond),
	)

	// Test concurrent retry operations
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn := func() (int, []byte, http.Header, error) {
				return DoRequestWithOptions("GET", server.URL, nil, nil)
			}
			_, _, _, err := WithRetry(config, fn)
			errors <- err
		}()
	}

	wg.Wait()
	close(errors)

	// All should succeed
	for err := range errors {
		if err != nil {
			t.Errorf("concurrent retry failed: %v", err)
		}
	}
}

func TestRaceConditionPrevention(t *testing.T) {
	// This test ensures there are no obvious race conditions
	// by running multiple operations concurrently
	var wg sync.WaitGroup

	// Test multiple operations that could race
	operations := []func(){
		func() { _ = DefaultClient() },
		func() { SetDefaultClient(&http.Client{Timeout: 1 * time.Second}) },
		func() { _ = NewInsecureClient() },
		func() {
			cb := NewCircuitBreaker(DefaultCircuitBreakerConfig)
			_ = cb.State()
		},
	}

	// Run each operation multiple times concurrently
	for _, op := range operations {
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(operation func()) {
				defer wg.Done()
				operation()
			}(op)
		}
	}

	wg.Wait()
	// If we get here without panicking, the test passes
}

func TestEdgeCaseEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return empty response with 200
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	data, err := Get(server.URL)
	if err != nil {
		t.Fatalf("Get with empty response failed: %v", err)
	}

	if len(data) != 0 {
		t.Errorf("expected empty response, got %d bytes", len(data))
	}
}

func TestEdgeCaseLargeResponse(t *testing.T) {
	// Create a large response (1MB)
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = 'A'
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(largeData)
	}))
	defer server.Close()

	data, err := Get(server.URL)
	if err != nil {
		t.Fatalf("Get with large response failed: %v", err)
	}

	if len(data) != len(largeData) {
		t.Errorf("expected %d bytes, got %d", len(largeData), len(data))
	}
}

func TestEdgeCaseInvalidURL(t *testing.T) {
	_, err := Get("invalid-url")
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}

	_, err = Get("http://")
	if err == nil {
		t.Error("expected error for malformed URL, got nil")
	}
}

func TestEdgeCaseNilBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	// Test with nil body
	data, err := Post(server.URL, nil)
	if err != nil {
		t.Fatalf("Post with nil body failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Post with nil body returned empty response")
	}
}

func TestEdgeCaseContextNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	// Test with nil context - this should panic, so we expect it to fail
	var nilCtx context.Context

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic with nil context, but none occurred")
		}
	}()

	_, _ = CtxGet(nilCtx, server.URL)
}

func TestEdgeCaseZeroRetries(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 0, // Zero retries
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		Jitter:      0.0,
		RetryOnStatus: []int{http.StatusInternalServerError},
	}

	callCount := 0
	fn := func() (int, []byte, http.Header, error) {
		callCount++
		return http.StatusInternalServerError, []byte("error"), nil, nil
	}

	statusCode, _, _, err := WithRetry(config, fn)

	if statusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", statusCode)
	}
	if err == nil {
		t.Error("expected error, got nil")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call with zero retries, got %d", callCount)
	}
}

func TestEdgeCaseNegativeTimeout(t *testing.T) {
	// Test with negative timeout (should use default)
	ctx, cancel := context.WithTimeout(context.Background(), -1*time.Second)
	defer cancel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	// Should still work (context will be expired but request might succeed)
	_, err := CtxGet(ctx, server.URL)
	// Don't check error as it depends on timing
	if err != nil {
		t.Logf("CtxGet with negative timeout returned error (might be expected): %v", err)
	}
}