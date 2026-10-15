package lock_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/tr1v3r/pkg/lock"
	redisdriver "github.com/tr1v3r/pkg/lock/driver/redis"
)

func ExampleLocker_TryLock() {
	client := goredis.NewClient(&goredis.Options{
		Addr:                  "127.0.0.1:6379",
		ContextTimeoutEnabled: true,
	})
	defer client.Close()
	driver, err := redisdriver.New(client, "example:locks:")
	if err != nil {
		panic(err)
	}
	locker, err := lock.New(driver, lock.WithAutoRenew())
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	lease, err := locker.TryLock(ctx, "resource:123", 10*time.Second)
	if errors.Is(err, lock.ErrNotAcquired) {
		return // another instance owns this lease
	}
	if err != nil {
		panic(err)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
		defer cancel()
		if releaseErr := lease.Release(cleanup); releaseErr != nil {
			fmt.Println("cleanup failed:", releaseErr)
		}
	}()

	// Pass lease.Context() to cooperative work. If ownership is lost, this
	// context is canceled independently of the renewal request's completion.
	_ = lease.Context()
	_ = lease.Deadline()
}
