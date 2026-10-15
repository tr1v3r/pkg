package lock

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"
)

// TryRun makes one attempt to reserve key for ttl and, if successful, calls fn
// synchronously with the original ctx. Contention returns false, nil. Once fn is
// invoked, executed is true and its error is returned unchanged.
//
// The reservation is never renewed or released, even when fn returns an error,
// panics, or ctx is canceled. WithAutoRenew has no effect. The TTL starts at
// acquisition, not callback completion; fn may outlive it, allowing another
// caller to run concurrently. This is a TTL execution gate, not a mutex or a
// guarantee of business idempotency. Only ctx governs callback cancellation.
//
// Acquisition uses WithOperationTimeout, but that timeout does not apply to fn.
// A canceled or expired acquisition does not invoke fn; any remote reservation
// is left to expire. No lease handle or background worker is created.
func (l *Locker) TryRun(
	ctx context.Context, key string, ttl time.Duration, fn func(context.Context) error,
) (bool, error) {
	if key == "" {
		return false, ErrInvalidKey
	}
	if ttl <= 0 {
		return false, ErrInvalidTTL
	}
	if fn == nil {
		return false, ErrNilFunc
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}

	owner := rand.Text()
	started := time.Now()
	request, cancel := context.WithTimeout(ctx, l.operationTimeout)
	defer cancel()
	acquired, acquireErr := l.driver.TryAcquire(request, key, owner, ttl)
	requestErr := request.Err()
	requestDeadline, _ := request.Deadline()
	cancel()
	if acquireErr != nil {
		return false, fmt.Errorf("lock: acquire %q: %w", key, acquireErr)
	}
	if contextErr := ctx.Err(); contextErr != nil {
		return false, contextErr
	}
	if requestErr != nil {
		return false, fmt.Errorf("lock: acquire %q: %w", key, requestErr)
	}
	now := time.Now()
	if !now.Before(requestDeadline) {
		return false, fmt.Errorf("lock: acquire %q: %w", key, context.DeadlineExceeded)
	}
	if !acquired {
		return false, nil
	}
	if !now.Before(started.Add(ttl)) {
		return false, ErrLeaseLost
	}
	return true, fn(ctx)
}

// Run retries contention until a reservation is acquired and fn is invoked, ctx
// is canceled, or a backend error occurs. Only false, nil results from TryRun are
// retried, using WithRetryInterval. Callback errors are never retried, regardless
// of their value. The executed result distinguishes callback errors from errors
// before execution. All other fixed-TTL semantics are the same as TryRun.
func (l *Locker) Run(
	ctx context.Context, key string, ttl time.Duration, fn func(context.Context) error,
) (bool, error) {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		executed, err := l.TryRun(ctx, key, ttl, fn)
		if executed || err != nil {
			return executed, err
		}
		timer.Reset(l.retryInterval)
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-timer.C:
		}
	}
}
