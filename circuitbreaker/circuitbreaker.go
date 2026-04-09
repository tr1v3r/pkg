package circuitbreaker

import (
	"fmt"
	"sync"
	"time"
)

// State represents the state of the circuit breaker
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern for protecting
// against cascading failures in distributed systems.
type CircuitBreaker struct {
	mu sync.Mutex

	state        State
	failureCount int32
	successCount int32
	lastFailure  time.Time

	// Configuration
	failureThreshold int32
	successThreshold int32
	openTimeout      time.Duration
}

// Config defines circuit breaker configuration
type Config struct {
	FailureThreshold int32         // Number of failures before opening circuit
	SuccessThreshold int32         // Number of successes before closing circuit
	OpenTimeout      time.Duration // How long to stay open before half-open
}

// DefaultConfig provides sensible defaults
var DefaultConfig = Config{
	FailureThreshold: 5,
	SuccessThreshold: 3,
	OpenTimeout:      30 * time.Second,
}

// New creates a new circuit breaker
func New(config Config) *CircuitBreaker {
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
		return &Error{Err: ErrOpen}
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
func (cb *CircuitBreaker) State() State {
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

// Error indicates the circuit breaker is open
type Error struct {
	Err error
}

func (e *Error) Error() string {
	return fmt.Sprintf("circuit breaker open: %v", e.Err)
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.Err
}

// ErrOpen is returned when the circuit breaker is open
var ErrOpen = fmt.Errorf("circuit breaker open")
