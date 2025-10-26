package fetch

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerStates(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig)

	// Initial state should be closed
	if cb.State() != StateClosed {
		t.Errorf("expected initial state to be closed, got %v", cb.State())
	}

	// Execute a successful request
	err := cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cb.State() != StateClosed {
		t.Errorf("expected state to remain closed after success, got %v", cb.State())
	}
}

func TestCircuitBreakerOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 1,
		OpenTimeout:      100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)

	// Fail enough times to open the circuit
	for i := 0; i < int(config.FailureThreshold); i++ {
		err := cb.Execute(func() error {
			return errors.New("failure")
		})
		if err == nil {
			t.Error("expected error, got nil")
		}
	}

	// Circuit should be open
	if cb.State() != StateOpen {
		t.Errorf("expected circuit to be open, got %v", cb.State())
	}

	// Should not allow execution when open
	if cb.Allow() {
		t.Error("expected circuit to not allow execution when open")
	}
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		OpenTimeout:      10 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)

	// Open the circuit
	for i := 0; i < int(config.FailureThreshold); i++ {
		cb.Execute(func() error {
			return errors.New("failure")
		})
	}

	// Circuit should be open
	if cb.State() != StateOpen {
		t.Errorf("expected circuit to be open, got %v", cb.State())
	}

	// Wait for circuit to become half-open
	time.Sleep(config.OpenTimeout + 10*time.Millisecond)

	// Calling Allow() should trigger transition to half-open
	if !cb.Allow() {
		t.Error("expected circuit to allow execution and transition to half-open")
	}

	// Circuit should be half-open after Allow() call
	if cb.State() != StateHalfOpen {
		t.Errorf("expected circuit to be half-open, got %v", cb.State())
	}

	// Succeed enough times to close the circuit
	for i := 0; i < int(config.SuccessThreshold); i++ {
		err := cb.Execute(func() error {
			return nil
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}

	// Circuit should be closed
	if cb.State() != StateClosed {
		t.Errorf("expected circuit to be closed, got %v", cb.State())
	}
}

func TestCircuitBreakerExecuteError(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 1,
		OpenTimeout:      100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)

	// Execute should return circuit breaker error when circuit is open
	cb.Execute(func() error {
		return errors.New("failure")
	})

	// Circuit should be open
	if cb.State() != StateOpen {
		t.Errorf("expected circuit to be open, got %v", cb.State())
	}

	// Execute should return CircuitBreakerError when circuit is open
	err := cb.Execute(func() error {
		return nil
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if _, ok := err.(*CircuitBreakerError); !ok {
		t.Errorf("expected CircuitBreakerError, got %T", err)
	}
}

func TestCircuitBreakerCounters(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		OpenTimeout:      100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config)

	// Execute some failures
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return errors.New("failure")
		})
	}

	if cb.FailureCount() != 2 {
		t.Errorf("expected failure count 2, got %d", cb.FailureCount())
	}

	// Execute some successes
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return nil
		})
	}

	if cb.SuccessCount() != 0 { // Success count should reset on state change
		t.Errorf("expected success count 0, got %d", cb.SuccessCount())
	}
	if cb.FailureCount() != 0 { // Failure count should reset on state change
		t.Errorf("expected failure count 0, got %d", cb.FailureCount())
	}
}