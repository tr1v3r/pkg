//go:build integration

package redis

import (
	"context"
	"crypto/rand"
	"os"
	"testing"
	"time"
)

// Run with PKG_LOCK_REDIS_ADDR=localhost:6379 go test -tags=integration ./lock/driver/redis.
// This test uses a unique namespace and only deletes its own key; never FLUSHDB.
func TestIntegrationRedisOwnershipAndExpiry(t *testing.T) {
	addr := os.Getenv("PKG_LOCK_REDIS_ADDR")
	if addr == "" {
		t.Skip("set PKG_LOCK_REDIS_ADDR to run real Redis integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	prefix := "pkg-lock-integration:" + rand.Text() + ":"
	first, client := newDriver(t, addr, prefix)
	second, _ := newDriver(t, addr, prefix)
	const key = "ownership"
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cleanupCancel()
		if err := client.Del(cleanupCtx, prefix+key).Err(); err != nil {
			t.Errorf("clean own integration key: %v", err)
		}
	})
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("connect to real Redis: %v", err)
	}

	ok, err := first.TryAcquire(ctx, key, "first", 5*time.Second)
	requireResult(t, true, ok, err)
	ok, err = second.TryAcquire(ctx, key, "second", time.Second)
	requireResult(t, false, ok, err)
	ok, err = second.Renew(ctx, key, "second", time.Minute)
	requireResult(t, false, ok, err)
	ok, err = second.Release(ctx, key, "second")
	requireResult(t, false, ok, err)
	ok, err = first.Renew(ctx, key, "first", 10*time.Second)
	requireResult(t, true, ok, err)
	if ttl, ttlErr := client.PTTL(ctx, prefix+key).Result(); ttlErr != nil || ttl < 5*time.Second {
		t.Fatalf("real renewal PTTL = %v, %v; want at least 5s", ttl, ttlErr)
	}
	// Shorten via the real owner-checked script, then observe Redis's real clock.
	ok, err = first.Renew(ctx, key, "first", 100*time.Millisecond)
	requireResult(t, true, ok, err)
	waitForRealExpiry(t, ctx, func() (bool, error) {
		count, existsErr := client.Exists(ctx, prefix+key).Result()
		return count == 0, existsErr
	})
	ok, err = first.Renew(ctx, key, "first", time.Second)
	requireResult(t, false, ok, err)
	ok, err = first.Release(ctx, key, "first")
	requireResult(t, false, ok, err)
	if count, existsErr := client.Exists(ctx, prefix+key).Result(); count != 0 || existsErr != nil {
		t.Fatalf("expired key was revived: %d, %v", count, existsErr)
	}

	ok, err = second.TryAcquire(ctx, key, "second", 5*time.Second)
	requireResult(t, true, ok, err)
	ok, err = first.Renew(ctx, key, "first", time.Minute)
	requireResult(t, false, ok, err)
	ok, err = first.Release(ctx, key, "first")
	requireResult(t, false, ok, err)
	if value, getErr := client.Get(ctx, prefix+key).Result(); value != "second" || getErr != nil {
		t.Fatalf("successor damaged by stale owner: %q, %v", value, getErr)
	}
	ok, err = second.Release(ctx, key, "second")
	requireResult(t, true, ok, err)
	ok, err = second.Release(ctx, key, "second")
	requireResult(t, false, ok, err)
}

func waitForRealExpiry(t *testing.T, ctx context.Context, expired func() (bool, error)) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(3 * time.Second)
	defer timeout.Stop()
	for {
		ok, err := expired()
		if err != nil {
			t.Fatalf("observe Redis expiry: %v", err)
		}
		if ok {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-timeout.C:
			t.Fatal("Redis lease did not expire within 3 seconds")
		case <-ticker.C:
		}
	}
}
