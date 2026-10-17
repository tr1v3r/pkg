package cache_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/tr1v3r/pkg/cache"
)

func ExampleNew() {
	c := cache.New[string, int]()
	defer c.Close() // Every cache owns a background goroutine.

	if err := c.Set("answer", 42, time.Hour); err != nil {
		panic(err)
	}
	value, ok := c.Get("answer")
	fmt.Println(value, ok)

	c.Delete("answer")
	value, ok = c.Get("answer")
	fmt.Println(value, ok)

	// Output:
	// 42 true
	// 0 false
}

func ExampleCache_Set() {
	c := cache.New[string, string]()
	defer c.Close()

	if err := c.Set("color", "blue", time.Hour); err != nil {
		panic(err)
	}
	// An invalid TTL leaves the existing value and deadline unchanged.
	err := c.Set("color", "red", 0)
	fmt.Println(errors.Is(err, cache.ErrInvalidTTL))
	value, ok := c.Get("color")
	fmt.Println(value, ok)

	// Output:
	// true
	// blue true
}

func ExampleCache_Get() {
	c := cache.New[string, int]()
	defer c.Close()

	if err := c.Set("short-lived", 42, time.Nanosecond); err != nil {
		panic(err)
	}
	time.Sleep(time.Millisecond)
	// Expired entries are misses, even if background cleanup has not run.
	value, ok := c.Get("short-lived")
	fmt.Println(value, ok)

	// Output:
	// 0 false
}

func ExampleCache_Close() {
	c := cache.New[string, int]()
	if err := c.Set("answer", 42, time.Hour); err != nil {
		panic(err)
	}

	c.Close()          // Clears entries and waits for the cleanup goroutine to exit.
	c.Close()          // Repeated calls are safe.
	c.Delete("answer") // A no-op after closing.
	value, ok := c.Get("answer")
	fmt.Println(value, ok)
	fmt.Println(errors.Is(c.Set("answer", 99, time.Hour), cache.ErrClosed))

	// Output:
	// 0 false
	// true
}

func ExampleOption() {
	seed := cache.Option[string, int](func(c *cache.Cache[string, int]) {
		if err := c.Set("answer", 42, time.Hour); err != nil {
			panic(err)
		}
	})
	c := cache.New[string, int](
		cache.WithCleanupBatchSize[string, int](64),
		cache.WithCleanupBatchBudget[string, int](time.Millisecond),
		seed,
	)
	defer c.Close()

	value, ok := c.Get("answer")
	fmt.Println(value, ok)

	// Output:
	// 42 true
}
