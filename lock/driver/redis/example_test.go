package redis_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	redisdriver "github.com/tr1v3r/pkg/lock/driver/redis"
)

// newExampleClient isolates each example. In production, replace miniredis with
// an application-owned Redis client and keep ContextTimeoutEnabled enabled.
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

func newExampleDriver() (*redisdriver.Driver, func()) {
	client, cleanup := newExampleClient()
	driver, err := redisdriver.New(client, "myapp:leases:")
	if err != nil {
		cleanup()
		panic(err)
	}
	return driver, cleanup
}

func ExampleNew() {
	client, cleanup := newExampleClient()
	defer cleanup()
	// Include a separator: the prefix is prepended verbatim to logical keys.
	driver, err := redisdriver.New(client, "myapp:leases:")
	if err != nil {
		panic(err)
	}
	fmt.Println("driver ready:", driver != nil)
	// New neither connects to Redis nor takes ownership of client resources.

	// Output:
	// driver ready: true
}

func ExampleDriver_TryAcquire() {
	driver, cleanup := newExampleDriver()
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// Direct driver users must generate a fresh owner for every new attempt.
	// The higher-level lock.Locker normally manages this token and lifecycle.
	owner := rand.Text()
	acquired, err := driver.TryAcquire(ctx, "resource:123", owner, time.Hour)
	fmt.Println("acquired:", acquired, "error:", err)

	// A new attempt cannot replace the live marker; use Renew to extend it.
	acquired, err = driver.TryAcquire(ctx, "resource:123", rand.Text(), time.Hour)
	fmt.Println("contended:", !acquired, "error:", err)
	renewed, err := driver.Renew(ctx, "resource:123", owner, time.Hour)
	fmt.Println("renewed:", renewed, "error:", err)
	released, err := driver.Release(ctx, "resource:123", owner)
	fmt.Println("released:", released, "error:", err)
	// Output:
	// acquired: true error: <nil>
	// contended: true error: <nil>
	// renewed: true error: <nil>
	// released: true error: <nil>
}

func ExampleDriver_Renew() {
	driver, cleanup := newExampleDriver()
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	owner := rand.Text()
	acquired, err := driver.TryAcquire(ctx, "resource:123", owner, time.Hour)
	if err != nil || !acquired {
		panic("failed to acquire initial owner")
	}
	renewed, err := driver.Renew(ctx, "resource:123", "not-the-owner", time.Hour)
	fmt.Println("other owner renewed:", renewed, "error:", err)
	renewed, err = driver.Renew(ctx, "resource:123", owner, time.Hour)
	fmt.Println("current owner renewed:", renewed, "error:", err)
	// Output:
	// other owner renewed: false error: <nil>
	// current owner renewed: true error: <nil>
}

func ExampleDriver_Release() {
	driver, cleanup := newExampleDriver()
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	owner := rand.Text()
	acquired, err := driver.TryAcquire(ctx, "resource:123", owner, time.Hour)
	if err != nil || !acquired {
		panic("failed to acquire initial owner")
	}
	// A stale or unrelated owner must never delete the current owner's key.
	released, err := driver.Release(ctx, "resource:123", "not-the-owner")
	fmt.Println("other owner released:", released, "error:", err)
	released, err = driver.Release(ctx, "resource:123", owner)
	fmt.Println("current owner released:", released, "error:", err)
	// Output:
	// other owner released: false error: <nil>
	// current owner released: true error: <nil>
}
