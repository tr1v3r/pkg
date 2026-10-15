//go:build go1.25

package lock

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

// Even an uncooperative transport cannot hold the LOCAL loss notification hostage.
// We unblock that transport explicitly so this test leaves no goroutine behind.
func TestLifecycleBlockedRenewCannotDelayLoss(t *testing.T) {
	for _, cancelEarly := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			resume := make(chan struct{})
			started := make(chan struct{})
			driver := hookedDriver{Driver: newMemoryDriver(), renew: func(context.Context, string, string, time.Duration) (bool, error) {
				close(started)
				<-resume
				return true, nil
			}}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			lease, err := testLocker(t, driver, WithAutoRenew()).TryLock(ctx, "k", 3*time.Second)
			if err != nil {
				t.Fatal(err)
			}
			synctest.Wait()
			time.Sleep(time.Second)
			synctest.Wait()
			<-started
			if cancelEarly {
				cancel()
			} else {
				time.Sleep(2 * time.Second)
			}
			synctest.Wait()
			select {
			case <-lease.done:
			default:
				t.Fatal("blocked renewal delayed lease loss notification")
			}
			cleanup, stop := context.WithTimeout(context.Background(), 100*time.Millisecond)
			result := make(chan error, 1)
			go func() { result <- lease.Release(cleanup) }()
			synctest.Wait()
			time.Sleep(100 * time.Millisecond)
			if err := <-result; !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal("Release wait ignored cleanup deadline", err)
			}
			stop()
			close(resume)
			synctest.Wait()
			select {
			case <-lease.stopped:
			default:
				t.Fatal("renew worker failed to drain")
			}
			if !errors.Is(lease.Err(), ErrLeaseLost) {
				t.Fatal("late successful renew revived abandoned handle", lease.Err())
			}
			_ = lease.Release(context.Background())
		})
	}
}

func TestAllDriverOperationsHaveFiniteDeadlines(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		memory := newMemoryDriver()
		const bound = 100 * time.Millisecond
		check := func(ctx context.Context) {
			t.Helper()
			deadline, ok := ctx.Deadline()
			if !ok || deadline.After(time.Now().Add(bound)) {
				t.Fatal("driver operation has no bounded deadline")
			}
		}
		driver := hookedDriver{
			acquire: func(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
				check(ctx)
				return memory.TryAcquire(ctx, key, owner, ttl)
			},
			renew: func(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
				check(ctx)
				return memory.Renew(ctx, key, owner, ttl)
			},
			release: func(ctx context.Context, key, owner string) (bool, error) {
				check(ctx)
				return memory.Release(ctx, key, owner)
			},
		}
		lease, err := testLocker(t, driver, WithOperationTimeout(bound)).TryLock(context.Background(), "k", time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		if err := lease.Renew(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := lease.Release(context.Background()); err != nil {
			t.Fatal(err)
		}
	})
}
