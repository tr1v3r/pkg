//go:build go1.25

package lock

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

func TestLifecycleDefaultExpiryWithoutRenewal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		locker := testLocker(t, newMemoryDriver())
		lease, err := locker.TryLock(context.Background(), "k", 3*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		time.Sleep(3 * time.Second)
		synctest.Wait()
		select {
		case <-lease.done:
		default:
			t.Fatal("expiry did not close Done")
		}
		if !errors.Is(lease.Err(), ErrLeaseLost) {
			t.Fatal(lease.Err())
		}
		if err := lease.Renew(context.Background()); !errors.Is(err, ErrLeaseLost) {
			t.Fatal("expired lease revived", err)
		}
		if err := lease.Release(context.Background()); !errors.Is(err, ErrLeaseLost) {
			t.Fatal(err)
		}
	})
}

func TestLifecycleManualRenewReschedulesExpiry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		locker := testLocker(t, newMemoryDriver())
		lease, err := locker.TryLock(context.Background(), "k", 3*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		time.Sleep(time.Second)
		if err := lease.Renew(context.Background()); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		time.Sleep(2 * time.Second)
		synctest.Wait()
		if lease.Err() != nil {
			t.Fatal("expired at original deadline", lease.Err())
		}
		time.Sleep(time.Second)
		synctest.Wait()
		if !errors.Is(lease.Err(), ErrLeaseLost) {
			t.Fatal("did not expire at renewed deadline", lease.Err())
		}
	})
}

func TestLifecycleAutoRenewAndCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		memory := newMemoryDriver()
		renewals := 0
		driver := hookedDriver{Driver: memory, renew: func(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
			renewals++
			return memory.Renew(ctx, key, owner, ttl)
		}}
		locker := testLocker(t, driver, WithAutoRenew())
		ctx, cancel := context.WithCancel(context.Background())
		lease, err := locker.TryLock(ctx, "k", 3*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		initial := lease.Deadline()
		synctest.Wait()
		time.Sleep(7 * time.Second)
		synctest.Wait()
		if renewals < 6 || lease.Err() != nil || !lease.Deadline().After(initial) {
			t.Fatal("automatic renewal did not preserve lease", renewals, lease.Err())
		}
		cancel()
		synctest.Wait()
		select {
		case <-lease.done:
		default:
			t.Fatal("cancellation did not signal lease loss")
		}
		if !errors.Is(lease.Err(), context.Canceled) {
			t.Fatal(lease.Err())
		}
		before := renewals
		time.Sleep(10 * time.Second)
		synctest.Wait()
		if renewals != before {
			t.Fatal("renewal worker survived cancellation")
		}
	})
}

func TestLifecycleAutoRenewFailure(t *testing.T) {
	failure := errors.New("renew backend failed")
	for _, backendErr := range []error{nil, failure} {
		synctest.Test(t, func(t *testing.T) {
			driver := hookedDriver{Driver: newMemoryDriver(), renew: func(context.Context, string, string, time.Duration) (bool, error) {
				return false, backendErr
			}}
			lease, err := testLocker(t, driver, WithAutoRenew()).TryLock(context.Background(), "k", 3*time.Second)
			if err != nil {
				t.Fatal(err)
			}
			synctest.Wait()
			time.Sleep(time.Second)
			synctest.Wait()
			select {
			case <-lease.done:
			default:
				t.Fatal("renewal failure not reported")
			}
			if !errors.Is(lease.Err(), ErrLeaseLost) || (backendErr != nil && !errors.Is(lease.Err(), backendErr)) {
				t.Fatal(lease.Err())
			}
		})
	}
}

func TestLifecycleSlowRequestsUseStartNotResponseTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		memory := newMemoryDriver()
		driver := hookedDriver{
			Driver: memory,
			acquire: func(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
				time.Sleep(time.Second)
				return memory.TryAcquire(ctx, key, owner, ttl)
			},
			renew: func(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
				time.Sleep(time.Second)
				return memory.Renew(ctx, key, owner, ttl)
			},
		}
		start := time.Now()
		lease, err := testLocker(t, driver, WithOperationTimeout(10*time.Second)).TryLock(context.Background(), "k", 3*time.Second)
		if err != nil || !lease.Deadline().Equal(start.Add(3*time.Second)) {
			t.Fatal("acquisition deadline overstated", err)
		}
		if err := lease.Renew(context.Background()); err != nil {
			t.Fatal(err)
		}
		if !lease.Deadline().Equal(start.Add(4 * time.Second)) {
			t.Fatal("renewal deadline overstated", lease.Deadline())
		}
		if err := lease.Release(context.Background()); err != nil {
			t.Fatal(err)
		}
	})
}

func TestLifecycleInflightRenewBoundedByOldDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		driver := hookedDriver{Driver: newMemoryDriver(), renew: func(ctx context.Context, _, _ string, _ time.Duration) (bool, error) {
			<-ctx.Done()
			return false, ctx.Err()
		}}
		lease, err := testLocker(t, driver, WithAutoRenew()).TryLock(context.Background(), "k", 3*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		time.Sleep(3 * time.Second)
		synctest.Wait()
		select {
		case <-lease.done:
		default:
			t.Fatal("in-flight renew outlived original deadline")
		}
		if !errors.Is(lease.Err(), ErrLeaseLost) {
			t.Fatal(lease.Err())
		}
	})
}

func TestLifecycleLateSuccessfulRenewCannotRevive(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		driver := hookedDriver{Driver: newMemoryDriver(), renew: func(context.Context, string, string, time.Duration) (bool, error) {
			time.Sleep(4 * time.Second)
			return true, nil
		}}
		lease, err := testLocker(t, driver).TryLock(context.Background(), "k", 3*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		if err := lease.Renew(context.Background()); !errors.Is(err, ErrLeaseLost) {
			t.Fatal("late renewal revived expired handle", err)
		}
		synctest.Wait()
	})
}

func TestLifecycleCanceledAndClosedGateWaits(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lease := testLease(t, testLocker(t, newMemoryDriver()), "k")
		<-lease.gate
		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan error, 1)
		go func() { result <- lease.Renew(ctx) }()
		synctest.Wait()
		cancel()
		if err := <-result; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		go func() { result <- lease.Renew(context.Background()) }()
		synctest.Wait()
		lease.finish(ErrClosed)
		if err := <-result; !errors.Is(err, ErrClosed) {
			t.Fatal(err)
		}
		lease.leave()
		if err := lease.Release(context.Background()); err != nil {
			t.Fatal(err)
		}
	})
}
