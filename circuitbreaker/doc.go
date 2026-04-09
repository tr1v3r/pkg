// Package circuitbreaker implements the circuit breaker pattern for protecting
// against cascading failures in distributed systems.
//
// A circuit breaker wraps potentially-failing operations and tracks consecutive
// failures. After a threshold is reached, the circuit "opens" and rejects
// further calls immediately (fast fail), giving the downstream service time to
// recover. After a configurable timeout, the circuit transitions to "half-open"
// and allows a limited number of probe requests to test recovery.
//
// Basic usage:
//
//	cb := circuitbreaker.New(circuitbreaker.Config{
//	    FailureThreshold: 5,
//	    SuccessThreshold: 3,
//	    OpenTimeout:      30 * time.Second,
//	})
//
//	err := cb.Execute(func() error {
//	    return callExternalService()
//	})
//
//	if errors.Is(err, circuitbreaker.ErrOpen) {
//	    // Handle fast-fail (circuit is open)
//	}
package circuitbreaker
