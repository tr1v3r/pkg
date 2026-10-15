//go:build integration

package lock_test

import (
	"context"
	"crypto/rand"
	"errors"
	"os"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/tr1v3r/pkg/lock"
	redisdriver "github.com/tr1v3r/pkg/lock/driver/redis"
)

func TestIntegrationRedisLeaseLifecycle(t *testing.T) {
	addr := os.Getenv("PKG_LOCK_REDIS_ADDR")
	if addr == "" {
		t.Skip("set PKG_LOCK_REDIS_ADDR to opt into real Redis tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := goredis.NewClient(&goredis.Options{
		Addr: addr, ContextTimeoutEnabled: true, MaxRetries: -1,
	})
	t.Cleanup(func() { _ = client.Close() })
	prefix := "pkg-lock-core-integration:" + rand.Text() + ":"
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), time.Second)
		defer stop()
		if cleanupErr := client.Del(cleanup, prefix+"resource", prefix+"auto").Err(); cleanupErr != nil {
			t.Errorf("clean own integration keys: %v", cleanupErr)
		}
	})
	backend, err := redisdriver.New(client, prefix)
	if err != nil {
		t.Fatal(err)
	}
	locker, err := lock.New(backend)
	if err != nil {
		t.Fatal(err)
	}
	other, err := lock.New(backend)
	if err != nil {
		t.Fatal(err)
	}
	lease, err := locker.TryLock(ctx, "resource", 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, busyErr := other.TryLock(ctx, "resource", time.Second); !errors.Is(busyErr, lock.ErrNotAcquired) {
		t.Fatal("cross-instance contention not enforced", busyErr)
	}
	if renewErr := lease.Renew(ctx); renewErr != nil {
		t.Fatal(renewErr)
	}
	if releaseErr := lease.Release(ctx); releaseErr != nil {
		t.Fatal(releaseErr)
	}
	successor, err := other.TryLock(ctx, "resource", 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if oldReleaseErr := lease.Release(ctx); oldReleaseErr != nil {
		t.Fatal(oldReleaseErr)
	}
	if successorErr := successor.Renew(ctx); successorErr != nil {
		t.Fatal("old handle interfered with successor", successorErr)
	}
	if cleanupErr := successor.Release(ctx); cleanupErr != nil {
		t.Fatal(cleanupErr)
	}

	automatic, err := lock.New(backend, lock.WithAutoRenew())
	if err != nil {
		t.Fatal(err)
	}
	lease, err = automatic.TryLock(ctx, "auto", 900*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Release(context.Background())
	initial := lease.Deadline()
	timer := time.NewTimer(time.Until(initial) + 300*time.Millisecond)
	defer timer.Stop()
	select {
	case <-lease.Done():
		t.Fatal("automatic lease expired", lease.Err())
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	case <-timer.C:
	}
	if !lease.Deadline().After(initial) || lease.Err() != nil {
		t.Fatal("automatic renewal failed", lease.Err())
	}
	if _, busyErr := other.TryLock(ctx, "auto", time.Second); !errors.Is(busyErr, lock.ErrNotAcquired) {
		t.Fatal("automatic renewal lost remote ownership", busyErr)
	}
	// Fault injection in our unique namespace: replace with a successor owner.
	if deleteErr := client.Del(ctx, prefix+"auto").Err(); deleteErr != nil {
		t.Fatal(deleteErr)
	}
	acquired, err := backend.TryAcquire(ctx, "auto", "successor", 3*time.Second)
	if err != nil || !acquired {
		t.Fatal("failed to install successor", acquired, err)
	}
	select {
	case <-lease.Done():
	case <-ctx.Done():
		t.Fatal("renewal did not report lost ownership", ctx.Err())
	}
	if !errors.Is(lease.Err(), lock.ErrLeaseLost) || lease.Context().Err() == nil {
		t.Fatal("lost lease/work context still live", lease.Err())
	}
	if staleErr := lease.Release(ctx); !errors.Is(staleErr, lock.ErrLeaseLost) {
		t.Fatal("stale cleanup should report lost ownership", staleErr)
	}
	value, err := client.Get(ctx, prefix+"auto").Result()
	if err != nil || value != "successor" {
		t.Fatal("stale handle changed successor", value, err)
	}
}
