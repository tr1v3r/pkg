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

func TestIntegrationRedisRunWindows(t *testing.T) {
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
	prefix := "pkg-lock-run-integration:" + rand.Text() + ":"
	keys := []string{
		prefix + "TryRun-success", prefix + "TryRun-failure",
		prefix + "Run-success", prefix + "Run-failure", prefix + "auto",
	}
	// Never touch other tests' namespaces or require an empty Redis database.
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), time.Second)
		defer stop()
		if cleanupErr := client.Del(cleanup, keys...).Err(); cleanupErr != nil {
			t.Errorf("clean up own Redis keys: %v", cleanupErr)
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
	other, err := lock.New(backend, lock.WithRetryInterval(5*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	const ttl = 180 * time.Millisecond
	failure := errors.New("callback failed")
	entries := map[string]func(context.Context, string, time.Duration, func(context.Context) error) (bool, error){
		"TryRun": locker.TryRun,
		"Run":    locker.Run,
	}
	for name, entry := range entries {
		for _, outcome := range []struct {
			name string
			err  error
		}{{"success", nil}, {"failure", failure}} {
			t.Run(name+"/"+outcome.name, func(t *testing.T) {
				key := name + "-" + outcome.name
				callbacks := 0
				started := time.Now()
				executed, runErr := entry(ctx, key, ttl, func(callbackCtx context.Context) error {
					callbacks++
					if callbackCtx != ctx {
						t.Error("callback context was replaced")
					}
					return outcome.err
				})
				if !executed || runErr != outcome.err || callbacks != 1 {
					t.Fatal("callback result changed", executed, runErr, callbacks)
				}
				owner, getErr := client.Get(ctx, prefix+key).Result()
				if getErr != nil || owner == "" {
					t.Fatal("callback completion removed Redis key", owner, getErr)
				}
				remaining, ttlErr := client.PTTL(ctx, prefix+key).Result()
				if ttlErr != nil || remaining <= 0 || remaining > ttl {
					t.Fatal("marker has no bounded TTL", remaining, ttlErr)
				}
				executed, runErr = other.TryRun(ctx, key, ttl, func(context.Context) error {
					t.Error("retained window admitted competing callback")
					return nil
				})
				if executed || runErr != nil {
					t.Fatal("contention should be (false, nil)", executed, runErr)
				}
				executed, runErr = other.Run(ctx, key, ttl, func(context.Context) error {
					callbacks++
					// Redis expiry uses integer milliseconds; exact boundaries are
					// covered separately with synctest's virtual clock.
					if time.Since(started) < ttl-5*time.Millisecond {
						t.Error("Run executed before original TTL elapsed")
					}
					return nil
				})
				if !executed || runErr != nil || callbacks != 2 {
					t.Fatal("Run did not execute after window expiry", executed, runErr, callbacks)
				}
				nextOwner, nextErr := client.Get(ctx, prefix+key).Result()
				if nextErr != nil || nextOwner == "" || nextOwner == owner {
					t.Fatal("successor marker missing or owner reused", nextOwner, nextErr)
				}
			})
		}
	}

	t.Run("auto renew does not extend execution window", func(t *testing.T) {
		automatic, newErr := lock.New(backend, lock.WithAutoRenew())
		if newErr != nil {
			t.Fatal(newErr)
		}
		executed, runErr := automatic.TryRun(ctx, "auto", ttl, func(callbackCtx context.Context) error {
			// Stay in the callback past both the renewal interval and the TTL.
			time.Sleep(ttl + ttl/2)
			if callbackCtx != ctx || callbackCtx.Err() != nil {
				t.Error("TTL canceled long callback", callbackCtx.Err())
			}
			return nil
		})
		if !executed || runErr != nil {
			t.Fatal(executed, runErr)
		}
		executed, runErr = other.TryRun(ctx, "auto", ttl, func(context.Context) error { return nil })
		if !executed || runErr != nil {
			t.Fatal("WithAutoRenew extended fixed execution window", executed, runErr)
		}
		if exists, existsErr := client.Exists(ctx, prefix+"auto").Result(); existsErr != nil || exists != 1 {
			t.Fatal("successor callback removed marker", exists, existsErr)
		}
	})
}
