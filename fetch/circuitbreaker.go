package fetch

import (
	"sync"
	"time"
)

// CircuitBreakerState represents the state of the circuit breaker
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker implements a simple circuit breaker pattern
type CircuitBreaker struct {
	mu sync.Mutex

	state        CircuitBreakerState
	failureCount int32
	successCount int32
	lastFailure  time.Time

	// Configuration
	failureThreshold int32
	successThreshold int32
	openTimeout      time.Duration
}

// CircuitBreakerConfig defines circuit breaker configuration
type CircuitBreakerConfig struct {
	FailureThreshold int32         // Number of failures before opening circuit
	SuccessThreshold int32         // Number of successes before closing circuit
	OpenTimeout      time.Duration // How long to stay open before half-open
}

// DefaultCircuitBreakerConfig provides sensible defaults
var DefaultCircuitBreakerConfig = CircuitBreakerConfig{
	FailureThreshold: 5,
	SuccessThreshold: 3,
	OpenTimeout:      30 * time.Second,
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: config.FailureThreshold,
		successThreshold: config.SuccessThreshold,
		openTimeout:      config.OpenTimeout,
	}
}

// Execute runs a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.Allow() {
		return &CircuitBreakerError{Err: ErrCircuitOpen}
	}

	err := fn()
	cb.RecordResult(err == nil)
	return err
}

// Allow checks if the circuit breaker allows execution
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateHalfOpen:
		return true
	case StateOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailure) > cb.openTimeout {
			cb.state = StateHalfOpen
			cb.successCount = 0
			return true
		}
		return false
	default:
		return false
	}
}

// RecordResult records the result of an execution
func (cb *CircuitBreaker) RecordResult(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if success {
		cb.onSuccess()
	} else {
		cb.onFailure()
	}
}

// onSuccess handles successful execution
func (cb *CircuitBreaker) onSuccess() {
	switch cb.state {
	case StateClosed:
		cb.failureCount = 0
	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.state = StateClosed
			cb.failureCount = 0
			cb.successCount = 0
		}
	}
}

// onFailure handles failed execution
func (cb *CircuitBreaker) onFailure() {
	switch cb.state {
	case StateClosed, StateHalfOpen:
		cb.failureCount++
		if cb.failureCount >= cb.failureThreshold {
			cb.state = StateOpen
			cb.lastFailure = time.Now()
		}
	}
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// FailureCount returns the current failure count
func (cb *CircuitBreaker) FailureCount() int32 {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.failureCount
}

// SuccessCount returns the current success count
func (cb *CircuitBreaker) SuccessCount() int32 {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.successCount
}
