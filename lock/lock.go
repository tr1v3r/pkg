package lock

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"reflect"
	"time"
)

var (
	ErrNotAcquired = errors.New("lock: lease not acquired")
	ErrLeaseLost   = errors.New("lock: lease lost")
	ErrClosed      = errors.New("lock: lease handle closed")
	ErrInvalidTTL  = errors.New("lock: TTL must be positive")
	ErrInvalidKey  = errors.New("lock: key must not be empty")
	ErrNilFunc     = errors.New("lock: callback must not be nil")
)

// Locker coordinates fixed-TTL execution gates and lease handles over a Driver.
// It is safe for concurrent use and owns neither connections nor background
// goroutines. Use New to construct it.
type Locker struct {
	driver           Driver
	retryInterval    time.Duration
	operationTimeout time.Duration
	autoRenew        bool
}

// Option configures a Locker before it is shared with other goroutines.
type Option func(*Locker) error

// WithRetryInterval sets the delay between contended Run/Lock attempts (default 50ms).
// Backend errors are returned immediately, not retried as contention.
func WithRetryInterval(interval time.Duration) Option {
	return func(l *Locker) error {
		if interval <= 0 {
			return errors.New("lock: retry interval must be positive")
		}
		l.retryInterval = interval
		return nil
	}
}

// WithOperationTimeout bounds each driver request (default 1s). An earlier caller
// deadline or remaining lease validity wins. Cancellation signals local loss
// promptly, but an in-flight transport may drain until this operation deadline.
func WithOperationTimeout(timeout time.Duration) Option {
	return func(l *Locker) error {
		if timeout <= 0 {
			return errors.New("lock: operation timeout must be positive")
		}
		l.operationTimeout = timeout
		return nil
	}
}

// WithAutoRenew enables renewal roughly one third of the TTL after each
// successful acquisition/renewal. It is off by default. Any renewal failure
// permanently invalidates the local handle and closes Lease.Done. This option
// only affects TryLock/Lock; TryRun/Run always use a fixed, unrenewed TTL.
func WithAutoRenew() Option {
	return func(l *Locker) error {
		l.autoRenew = true
		return nil
	}
}

// New constructs a Locker. A driver (including its underlying value) must be
// non-nil. Options are applied in order; nil options are rejected.
func New(driver Driver, options ...Option) (*Locker, error) {
	if driver == nil || nilValue(driver) {
		return nil, errors.New("lock: driver must not be nil")
	}
	l := &Locker{driver: driver, retryInterval: 50 * time.Millisecond, operationTimeout: time.Second}
	for _, option := range options {
		if option == nil {
			return nil, errors.New("lock: option must not be nil")
		}
		if err := option(l); err != nil {
			return nil, err
		}
	}
	return l, nil
}

func nilValue(value any) bool {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

// TryLock makes one acquisition attempt. Each attempt has a fresh owner token;
// contention returns ErrNotAcquired and backend errors retain their identity.
// The context governs both acquisition and the returned lease's lifetime.
// Cancellation stops renewal but does not automatically delete the remote key.
//
// Local validity starts BEFORE the driver request, not at its response. A slow
// successful acquisition that outlives its TTL or request deadline is rejected and
// best-effort released.
// An errored acquisition may have succeeded remotely; its key is left to expire.
func (l *Locker) TryLock(ctx context.Context, key string, ttl time.Duration) (*Lease, error) {
	if key == "" {
		return nil, ErrInvalidKey
	}
	if ttl <= 0 {
		return nil, ErrInvalidTTL
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	owner := rand.Text()
	started := time.Now()
	request, cancel := context.WithTimeout(ctx, l.operationTimeout)
	defer cancel()
	acquired, err := l.driver.TryAcquire(request, key, owner, ttl)
	if err != nil {
		return nil, fmt.Errorf("lock: acquire %q: %w", key, err)
	}
	if requestErr := requestError(request); requestErr != nil {
		err := fmt.Errorf("lock: acquire %q: %w", key, requestErr)
		if !acquired {
			return nil, err
		}
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), l.operationTimeout)
		defer cancel()
		_, cleanupErr := l.driver.Release(cleanup, key, owner)
		return nil, errors.Join(ErrLeaseLost, err, cleanupErr)
	}
	if !acquired {
		return nil, ErrNotAcquired
	}
	lease := newLease(ctx, l, key, owner, ttl, started.Add(ttl))
	if err := lease.Err(); err != nil {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), l.operationTimeout)
		defer cancel()
		return nil, errors.Join(err, lease.Release(cleanup))
	}
	return lease, nil
}

// Lock retries contention until acquired or ctx is canceled. It does not retry
// backend errors or turn an uncertain acquisition into a successful one.
func (l *Locker) Lock(ctx context.Context, key string, ttl time.Duration) (*Lease, error) {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		lease, err := l.TryLock(ctx, key, ttl)
		if !errors.Is(err, ErrNotAcquired) {
			return lease, err
		}
		timer.Reset(l.retryInterval)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

// requestError also checks the clock: a deadline can pass before the context's
// timer goroutine has recorded cancellation.
func requestError(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if deadline, ok := ctx.Deadline(); ok && !time.Now().Before(deadline) {
		return context.DeadlineExceeded
	}
	return nil
}
