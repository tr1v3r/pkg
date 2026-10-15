//go:build go1.25

package cache

import (
	"testing"
	"testing/synctest"
	"time"
)

func TestSchedulerFirstInsertionWakesIdleWorker(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New[string, int]()
		defer c.Close()
		synctest.Wait() // The empty-cache worker is waiting without a timer.

		schedulerSet(t, c, "first", 1, time.Minute)
		synctest.Wait()
		time.Sleep(time.Minute - time.Nanosecond)
		synctest.Wait()
		schedulerAssertKeys(t, c, "first")
		schedulerAssertGet(t, c, "first", 1, true)

		time.Sleep(time.Nanosecond)
		synctest.Wait()
		schedulerAssertKeys(t, c)
	})
}

func TestSchedulerEarlierInsertionResetsTimer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New[string, int]()
		defer c.Close()
		schedulerSet(t, c, "later", 1, time.Hour)
		synctest.Wait()
		time.Sleep(time.Minute)

		schedulerSet(t, c, "earlier", 2, time.Minute)
		synctest.Wait()
		time.Sleep(time.Minute - time.Nanosecond)
		synctest.Wait()
		schedulerAssertKeys(t, c, "earlier", "later")

		time.Sleep(time.Nanosecond)
		synctest.Wait()
		schedulerAssertKeys(t, c, "later")
		schedulerAssertGet(t, c, "later", 1, true)

		time.Sleep(time.Hour - 2*time.Minute)
		synctest.Wait()
		schedulerAssertKeys(t, c)
	})
}

func TestSchedulerSuccessiveInsertionsResetActiveTimer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New[string, int]()
		defer c.Close()
		schedulerSet(t, c, "a", 1, 30*time.Second)
		synctest.Wait()
		time.Sleep(5 * time.Second)

		schedulerSet(t, c, "b", 2, 20*time.Second)
		synctest.Wait()
		time.Sleep(3 * time.Second)

		schedulerSet(t, c, "c", 3, 10*time.Second)
		synctest.Wait()
		time.Sleep(10*time.Second - time.Nanosecond)
		synctest.Wait()
		schedulerAssertKeys(t, c, "a", "b", "c")
		time.Sleep(time.Nanosecond)
		synctest.Wait()
		schedulerAssertKeys(t, c, "a", "b")

		time.Sleep(7 * time.Second)
		synctest.Wait()
		schedulerAssertKeys(t, c, "a")
		time.Sleep(5 * time.Second)
		synctest.Wait()
		schedulerAssertKeys(t, c)
	})
}

func TestSchedulerEmptyAfterDeleteRearmsOnInsertion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New[string, int]()
		defer c.Close()
		schedulerSet(t, c, "deleted", 1, time.Minute)
		synctest.Wait()
		time.Sleep(30 * time.Second)
		c.Delete("deleted")
		synctest.Wait()
		schedulerAssertKeys(t, c)

		// Stay idle beyond the cancelled deadline, then start a new lifetime.
		time.Sleep(time.Hour)
		synctest.Wait()
		schedulerSet(t, c, "new", 2, time.Minute)
		synctest.Wait()
		time.Sleep(time.Minute - time.Nanosecond)
		synctest.Wait()
		schedulerAssertKeys(t, c, "new")
		schedulerAssertGet(t, c, "new", 2, true)
		time.Sleep(time.Nanosecond)
		synctest.Wait()
		schedulerAssertKeys(t, c)
	})
}

func TestSchedulerRefreshExtendsDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New[string, int]()
		defer c.Close()
		schedulerSet(t, c, "refreshed", 1, time.Minute)
		schedulerSet(t, c, "other", 2, 2*time.Minute)
		synctest.Wait()
		time.Sleep(30 * time.Second)

		schedulerSet(t, c, "refreshed", 3, 3*time.Minute)
		synctest.Wait()
		schedulerAssertKeys(t, c, "refreshed", "other")
		time.Sleep(30 * time.Second) // The refreshed entry's original deadline.
		synctest.Wait()
		schedulerAssertKeys(t, c, "refreshed", "other")
		schedulerAssertGet(t, c, "refreshed", 3, true)

		time.Sleep(time.Minute)
		synctest.Wait()
		schedulerAssertKeys(t, c, "refreshed")
		time.Sleep(90*time.Second - time.Nanosecond)
		synctest.Wait()
		schedulerAssertKeys(t, c, "refreshed")
		time.Sleep(time.Nanosecond)
		synctest.Wait()
		schedulerAssertKeys(t, c)
	})
}

func TestSchedulerRefreshShortensDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New[string, int]()
		defer c.Close()
		schedulerSet(t, c, "refreshed", 1, time.Hour)
		schedulerSet(t, c, "other", 2, 30*time.Minute)
		synctest.Wait()
		time.Sleep(time.Minute)

		schedulerSet(t, c, "refreshed", 3, time.Minute)
		synctest.Wait()
		time.Sleep(time.Minute - time.Nanosecond)
		synctest.Wait()
		schedulerAssertKeys(t, c, "refreshed", "other")
		schedulerAssertGet(t, c, "refreshed", 3, true)
		time.Sleep(time.Nanosecond)
		synctest.Wait()
		schedulerAssertKeys(t, c, "other")

		// A new lifetime must also survive the first lifetime's old timer.
		schedulerSet(t, c, "refreshed", 4, 2*time.Hour)
		synctest.Wait()
		time.Sleep(58 * time.Minute)
		synctest.Wait()
		schedulerAssertKeys(t, c, "refreshed")
		schedulerAssertGet(t, c, "refreshed", 4, true)
		time.Sleep(62 * time.Minute)
		synctest.Wait()
		schedulerAssertKeys(t, c)
	})
}

func TestSchedulerGetMissesExactlyAtDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New[string, int]()
		defer c.Close()
		synctest.Wait()

		// Seed without notifying the idle worker: Get must enforce the exact
		// deadline even when the expired entry has not been physically removed.
		item := &entry[string, int]{key: "boundary", value: 7, deadline: time.Now().Add(time.Minute)}
		c.mu.Lock()
		c.items[item.key] = item
		c.expirations = append(c.expirations, item)
		c.mu.Unlock()

		time.Sleep(time.Minute - time.Nanosecond)
		schedulerAssertGet(t, c, "boundary", 7, true)
		time.Sleep(time.Nanosecond)
		schedulerAssertKeys(t, c, "boundary")
		schedulerAssertGet(t, c, "boundary", 0, false)

		// An ordinary Set wakes the worker and must physically clean the fixture.
		schedulerSet(t, c, "wake", 8, time.Hour)
		synctest.Wait()
		schedulerAssertKeys(t, c, "wake")
	})
}

func TestSchedulerDeleteAndReinsertIgnoresOldDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New[string, int]()
		defer c.Close()
		schedulerSet(t, c, "key", 1, time.Minute)
		synctest.Wait()
		time.Sleep(30 * time.Second)
		c.Delete("key")
		schedulerAssertKeys(t, c)
		schedulerSet(t, c, "key", 2, 2*time.Minute)
		synctest.Wait()

		time.Sleep(30 * time.Second)
		synctest.Wait()
		schedulerAssertKeys(t, c, "key")
		schedulerAssertGet(t, c, "key", 2, true)
		time.Sleep(90*time.Second - time.Nanosecond)
		synctest.Wait()
		schedulerAssertKeys(t, c, "key")
		time.Sleep(time.Nanosecond)
		synctest.Wait()
		schedulerAssertKeys(t, c)
	})
}

func TestSchedulerDeleteEarliestKeepsNextDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New[string, int]()
		defer c.Close()
		schedulerSet(t, c, "first", 1, time.Minute)
		schedulerSet(t, c, "second", 2, 2*time.Minute)
		synctest.Wait()
		time.Sleep(30 * time.Second)
		c.Delete("first")
		synctest.Wait()
		schedulerAssertKeys(t, c, "second")

		time.Sleep(30 * time.Second)
		synctest.Wait()
		schedulerAssertKeys(t, c, "second")
		time.Sleep(time.Minute - time.Nanosecond)
		synctest.Wait()
		schedulerAssertKeys(t, c, "second")
		time.Sleep(time.Nanosecond)
		synctest.Wait()
		schedulerAssertKeys(t, c)
	})
}

func TestSchedulerBatchWithIdenticalDeadlines(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New[int, int]()
		defer c.Close()
		const count = 128
		for key := range count {
			if err := c.Set(key, key, time.Minute); err != nil {
				t.Fatalf("Set(%d): %v", key, err)
			}
		}
		synctest.Wait()
		time.Sleep(time.Minute - time.Nanosecond)
		synctest.Wait()
		c.mu.RLock()
		items, expirations := len(c.items), len(c.expirations)
		c.mu.RUnlock()
		if items != count || expirations != count {
			t.Fatalf("before deadline: items = %d, expirations = %d, want %d each", items, expirations, count)
		}

		time.Sleep(time.Nanosecond)
		synctest.Wait()
		c.mu.RLock()
		items, expirations = len(c.items), len(c.expirations)
		c.mu.RUnlock()
		if items != 0 || expirations != 0 {
			t.Fatalf("at deadline: items = %d, expirations = %d, want 0 each", items, expirations)
		}
	})
}

func TestSchedulerCloseStopsWorker(t *testing.T) {
	for _, tc := range []struct {
		name string
		ttl  time.Duration
	}{
		{name: "empty"},
		{name: "nonempty", ttl: time.Minute},
		{name: "long_ttl", ttl: 100 * 365 * 24 * time.Hour},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				c := New[string, int]()
				defer c.Close()
				if tc.ttl > 0 {
					schedulerSet(t, c, "key", 1, tc.ttl)
				}
				synctest.Wait()
				started := time.Now()
				c.Close()
				synctest.Wait()
				if elapsed := time.Since(started); elapsed != 0 {
					t.Fatalf("Close advanced the clock by %v, want immediate shutdown", elapsed)
				}
				schedulerAssertClosed(t, c)
				c.Close()
				schedulerAssertClosed(t, c)
			})
		})
	}
}

func TestSchedulerConcurrentCloseStopsWorker(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New[string, int]()
		defer c.Close()
		schedulerSet(t, c, "key", 1, 100*365*24*time.Hour)
		synctest.Wait()
		const callers = 32
		start := make(chan struct{})
		returned := make(chan struct{}, callers)
		for range callers {
			go func() {
				<-start
				c.Close()
				returned <- struct{}{}
			}()
		}
		started := time.Now()
		close(start)
		synctest.Wait()
		if got := len(returned); got != callers {
			t.Fatalf("Close returned for %d callers, want %d", got, callers)
		}
		if elapsed := time.Since(started); elapsed != 0 {
			t.Fatalf("concurrent Close advanced the clock by %v", elapsed)
		}
		schedulerAssertClosed(t, c)
	})
}

func schedulerSet(t *testing.T, c *Cache[string, int], key string, value int, ttl time.Duration) {
	t.Helper()
	if err := c.Set(key, value, ttl); err != nil {
		t.Fatalf("Set(%q): %v", key, err)
	}
}

func schedulerAssertGet(t *testing.T, c *Cache[string, int], key string, want int, wantOK bool) {
	t.Helper()
	if got, ok := c.Get(key); got != want || ok != wantOK {
		t.Fatalf("Get(%q) = (%d, %t), want (%d, %t)", key, got, ok, want, wantOK)
	}
}

func schedulerAssertKeys(t *testing.T, c *Cache[string, int], keys ...string) {
	t.Helper()
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.items) != len(keys) || len(c.expirations) != len(keys) {
		t.Fatalf("items = %d, expirations = %d, want keys %v", len(c.items), len(c.expirations), keys)
	}
	for _, key := range keys {
		if _, ok := c.items[key]; !ok {
			t.Fatalf("key %q missing from physical storage; want keys %v", key, keys)
		}
	}
	for index, item := range c.expirations {
		if item.index != index || c.items[item.key] != item {
			t.Fatalf("expiration %d is not indexed consistently in heap and items", index)
		}
	}
}

func schedulerAssertClosed(t *testing.T, c *Cache[string, int]) {
	t.Helper()
	select {
	case <-c.done:
	default:
		t.Fatal("Close returned before the worker exited")
	}
	schedulerAssertKeys(t, c)
	for _, ttl := range []time.Duration{time.Hour, 0, -time.Second} {
		if err := c.Set("key", 2, ttl); err != ErrClosed {
			t.Fatalf("Set after Close with TTL %v = %v, want ErrClosed", ttl, err)
		}
	}
	schedulerAssertGet(t, c, "key", 0, false)
	c.Delete("key")
	c.Delete("missing")
	schedulerAssertKeys(t, c)
}
