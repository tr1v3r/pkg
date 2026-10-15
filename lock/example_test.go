package lock_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/tr1v3r/pkg/lock"
	redisdriver "github.com/tr1v3r/pkg/lock/driver/redis"
)

// newExampleClient gives each example its own Redis server and connections.
// In production, replace miniredis with an application-owned Redis client;
// keep ContextTimeoutEnabled enabled and close the client on shutdown.
func newExampleClient() (*goredis.Client, func()) {
	server, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	client := goredis.NewClient(&goredis.Options{
		Addr: server.Addr(), ContextTimeoutEnabled: true, MaxRetries: -1,
	})
	return client, func() {
		closeErr := client.Close()
		server.Close()
		if closeErr != nil {
			panic(closeErr)
		}
	}
}

func newExampleLocker(options ...lock.Option) (*lock.Locker, func()) {
	client, cleanup := newExampleClient()
	driver, err := redisdriver.New(client, "example:leases:")
	if err != nil {
		cleanup()
		panic(err)
	}
	locker, err := lock.New(driver, options...)
	if err != nil {
		cleanup()
		panic(err)
	}
	return locker, cleanup
}

// releaseExampleLease stops all lease workers before the Redis client is closed.
// Cleanup needs its own live context, not the possibly canceled work context.
func releaseExampleLease(lease *lock.Lease) {
	cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := lease.Release(cleanup); err != nil {
		panic(err)
	}
}

func ExampleNew() {
	client, cleanup := newExampleClient()
	defer cleanup()

	// Inject a backend; the core package does not own Redis connections.
	driver, err := redisdriver.New(client, "myapp:gates:")
	if err != nil {
		panic(err)
	}
	locker, err := lock.New(driver)
	if err != nil {
		panic(err)
	}
	executed, err := locker.TryRun(context.Background(), "event:123", time.Hour,
		func(context.Context) error { return nil })
	fmt.Println("executed:", executed, "error:", err)
	// Output:
	// executed: true error: <nil>
}

func ExampleWithRetryInterval() {
	// Only contention is retried, with this delay between attempts.
	locker, cleanup := newExampleLocker(lock.WithRetryInterval(20 * time.Millisecond))
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	executed, err := locker.Run(ctx, "job:123", time.Hour, func(context.Context) error { return nil })
	fmt.Println("executed:", executed, "error:", err)
	// Output:
	// executed: true error: <nil>
}

func ExampleWithOperationTimeout() {
	// Each backend request is bounded; the callback still receives the caller ctx.
	locker, cleanup := newExampleLocker(lock.WithOperationTimeout(500 * time.Millisecond))
	defer cleanup()
	ctx := context.Background()
	executed, err := locker.TryRun(ctx, "job:123", time.Hour, func(callbackCtx context.Context) error {
		fmt.Println("original caller context:", callbackCtx == ctx)
		return nil
	})
	fmt.Println("executed:", executed, "error:", err)
	// Output:
	// original caller context: true
	// executed: true error: <nil>
}

func ExampleWithAutoRenew() {
	// Auto-renew is opt-in for leases; it has no effect on TryRun or Run gates.
	locker, cleanup := newExampleLocker(lock.WithAutoRenew())
	defer cleanup()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lease, err := locker.TryLock(ctx, "resource:123", time.Hour)
	if err != nil {
		panic(err)
	}
	defer releaseExampleLease(lease)

	// Cooperative work must stop when the lease's context is canceled.
	work := lease.Context()
	fmt.Println("work active:", work.Err() == nil)
	cancel()
	<-work.Done()
	fmt.Println("work canceled:", errors.Is(work.Err(), context.Canceled))
	// Output:
	// work active: true
	// work canceled: true
}

func ExampleLocker_TryRun() {
	locker, cleanup := newExampleLocker()
	defer cleanup()
	process := func(context.Context) error {
		fmt.Println("process event:123")
		return nil
	}
	executed, err := locker.TryRun(context.Background(), "event:123", time.Hour, process)
	fmt.Println("first:", executed, err)
	executed, err = locker.TryRun(context.Background(), "event:123", time.Hour, process)
	fmt.Println("duplicate:", executed, err)
	// There is no Release: the marker remains until the original TTL expires.

	// Output:
	// process event:123
	// first: true <nil>
	// duplicate: false <nil>
}

func ExampleLocker_TryRun_callbackError() {
	locker, cleanup := newExampleLocker()
	defer cleanup()
	failure := errors.New("processing failed")
	executed, err := locker.TryRun(context.Background(), "event:123", time.Hour,
		func(context.Context) error { return failure })
	fmt.Println("callback executed:", executed, "original error:", err == failure)

	// An error does not delete the marker or retry the callback.
	executed, err = locker.TryRun(context.Background(), "event:123", time.Hour,
		func(context.Context) error { panic("duplicate callback must not run") })
	fmt.Println("duplicate:", executed, err)
	// Output:
	// callback executed: true original error: true
	// duplicate: false <nil>
}

func ExampleLocker_Run() {
	locker, cleanup := newExampleLocker(lock.WithRetryInterval(20 * time.Millisecond))
	defer cleanup()
	// A caller deadline also bounds waiting for a contended window to expire.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	executed, err := locker.Run(ctx, "job:123", time.Hour, func(work context.Context) error {
		fmt.Println("job context active:", work.Err() == nil)
		return nil
	})
	fmt.Println("executed:", executed, "error:", err)
	// Output:
	// job context active: true
	// executed: true error: <nil>
}

func ExampleLocker_Run_cancellation() {
	locker, cleanup := newExampleLocker()
	defer cleanup()
	executed, err := locker.TryRun(context.Background(), "job:123", time.Hour,
		func(context.Context) error { return nil })
	if err != nil || !executed {
		panic("failed to reserve initial window")
	}
	// Bound the wait for a busy key instead of waiting for the hour-long marker.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	executed, err = locker.Run(ctx, "job:123", time.Hour,
		func(context.Context) error { panic("canceled callback must not run") })
	fmt.Println("executed:", executed, "timed out:", errors.Is(err, context.DeadlineExceeded))
	// Output:
	// executed: false timed out: true
}

func ExampleLocker_TryLock() {
	locker, cleanup := newExampleLocker()
	defer cleanup()
	lease, err := locker.TryLock(context.Background(), "resource:123", time.Hour)
	if err != nil {
		panic(err)
	}
	defer releaseExampleLease(lease)
	fmt.Println("acquired:", lease.Key())

	_, err = locker.TryLock(context.Background(), "resource:123", time.Hour)
	fmt.Println("contended:", errors.Is(err, lock.ErrNotAcquired))
	// Output:
	// acquired: resource:123
	// contended: true
}

func ExampleLocker_Lock() {
	locker, cleanup := newExampleLocker(lock.WithRetryInterval(20 * time.Millisecond))
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	lease, err := locker.Lock(ctx, "resource:123", time.Hour)
	if err != nil {
		panic(err)
	}
	defer releaseExampleLease(lease)
	fmt.Println("acquired:", lease.Key())
	// Output:
	// acquired: resource:123
}

func ExampleLease_Renew() {
	locker, cleanup := newExampleLocker()
	defer cleanup()
	lease, err := locker.TryLock(context.Background(), "resource:123", time.Hour)
	if err != nil {
		panic(err)
	}
	defer releaseExampleLease(lease)

	// Manual renewal extends ownership by the original TTL, not a new duration.
	err = lease.Renew(context.Background())
	fmt.Println("renewed:", err == nil)
	// Output:
	// renewed: true
}

func ExampleLease_Release() {
	locker, cleanup := newExampleLocker()
	defer cleanup()
	lease, err := locker.TryLock(context.Background(), "resource:123", time.Hour)
	if err != nil {
		panic(err)
	}
	defer releaseExampleLease(lease)
	fmt.Println("release:", lease.Release(context.Background()))
	fmt.Println("release again:", lease.Release(context.Background()))
	// Output:
	// release: <nil>
	// release again: <nil>
}

func ExampleLease_Key() {
	locker, cleanup := newExampleLocker()
	defer cleanup()
	lease, err := locker.TryLock(context.Background(), "resource:123", time.Hour)
	if err != nil {
		panic(err)
	}
	defer releaseExampleLease(lease)
	// The logical key does not include the Redis namespace prefix.
	fmt.Println(lease.Key())
	// Output:
	// resource:123
}

func ExampleLease_Deadline() {
	locker, cleanup := newExampleLocker()
	defer cleanup()
	lease, err := locker.TryLock(context.Background(), "resource:123", time.Hour)
	if err != nil {
		panic(err)
	}
	defer releaseExampleLease(lease)
	// This is a conservative local deadline, not proof of remote ownership.
	fmt.Println("validity remaining:", time.Until(lease.Deadline()) > 0)
	// Output:
	// validity remaining: true
}

func ExampleLease_Done() {
	locker, cleanup := newExampleLocker()
	defer cleanup()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lease, err := locker.TryLock(ctx, "resource:123", time.Hour)
	if err != nil {
		panic(err)
	}
	defer releaseExampleLease(lease)

	cancel()
	<-lease.Done() // Stop protected work; this does not imply remote deletion.
	fmt.Println("caller canceled:", errors.Is(lease.Err(), context.Canceled))
	// Output:
	// caller canceled: true
}

func ExampleLease_Err() {
	locker, cleanup := newExampleLocker()
	defer cleanup()
	lease, err := locker.TryLock(context.Background(), "resource:123", time.Hour)
	if err != nil {
		panic(err)
	}
	defer releaseExampleLease(lease)
	fmt.Println("initial error:", lease.Err())
	if releaseErr := lease.Release(context.Background()); releaseErr != nil {
		panic(releaseErr)
	}
	fmt.Println("closed:", errors.Is(lease.Err(), lock.ErrClosed))
	// Output:
	// initial error: <nil>
	// closed: true
}

func ExampleLease_Context() {
	locker, cleanup := newExampleLocker()
	defer cleanup()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lease, err := locker.TryLock(ctx, "resource:123", time.Hour)
	if err != nil {
		panic(err)
	}
	defer releaseExampleLease(lease)

	// Pass this context to I/O or other cooperative work protected by the lease.
	work := lease.Context()
	fmt.Println("work active:", work.Err() == nil)
	cancel()
	<-work.Done()
	fmt.Println("work canceled:", errors.Is(work.Err(), context.Canceled))
	// Output:
	// work active: true
	// work canceled: true
}
