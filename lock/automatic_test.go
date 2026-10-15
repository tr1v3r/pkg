package lock

import (
	"context"
	"errors"
	"testing"
	"time"
)

// These smoke tests also run on the minimum Go1.24 toolchain. Exact timing and
// fault scenarios are covered by virtual time on Go1.25+.
func TestAutomaticRenewMinimumToolchain(t *testing.T) {
	memory := newMemoryDriver()
	renewed := make(chan struct{}, 1)
	driver := hookedDriver{Driver: memory, renew: func(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
		owned, err := memory.Renew(ctx, key, owner, ttl)
		select {
		case renewed <- struct{}{}:
		default:
		}
		return owned, err
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	lease, err := testLocker(t, driver, WithAutoRenew()).TryLock(ctx, "k", 600*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Release(context.Background())
	select {
	case <-renewed:
	case <-lease.done:
		t.Fatal("lease invalidated before automatic renewal", lease.Err())
	case <-ctx.Done():
		t.Fatal("automatic renewal did not run", ctx.Err())
	}
	if lease.Context().Err() != nil {
		t.Fatal("work context invalidated before release", lease.Context().Err())
	}
	if err := lease.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	waitClosed(t, lease.Context().Done())
	waitClosed(t, lease.stopped)
}

func TestDefaultExpiryMinimumToolchain(t *testing.T) {
	lease, err := testLocker(t, newMemoryDriver()).TryLock(context.Background(), "k", 40*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	waitClosed(t, lease.done)
	waitClosed(t, lease.stopped)
	if !errors.Is(lease.Err(), ErrLeaseLost) {
		t.Fatal(lease.Err())
	}
}

func TestCanceledContendedAcquisition(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	driver := hookedDriver{acquire: func(context.Context, string, string, time.Duration) (bool, error) {
		cancel()
		return false, nil
	}}
	if _, err := testLocker(t, driver).TryLock(ctx, "k", time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
