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
		c.items.Store(item.key, item)
		c.expirations = append(c.expirations, item)
		c.mu.Unlock()

		time.Sleep(time.Minute - time.Nanosecond)
		schedulerAssertGet(t, c, "boundary", 7, true)
		time.Sleep(time.Nanosecond)
		schedulerAssertKeys(t, c, "boundary")
		schedulerAssertGet(t, c, "boundary", 0, false)

		// Deleting the earliest entry changes the schedule and wakes the worker.
		c.Delete("boundary")
		synctest.Wait()
		schedulerAssertKeys(t, c)
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
		c.mu.Lock()
		items, expirations := itemCount(c), len(c.expirations)
		c.mu.Unlock()
		if items != count || expirations != count {
			t.Fatalf("before deadline: items = %d, expirations = %d, want %d each", items, expirations, count)
		}

		time.Sleep(time.Nanosecond)
		synctest.Wait()
		c.mu.Lock()
		items, expirations = itemCount(c), len(c.expirations)
		c.mu.Unlock()
		if items != 0 || expirations != 0 {
			t.Fatalf("at deadline: items = %d, expirations = %d, want 0 each", items, expirations)
		}
	})
}

func TestSchedulerBatchedCleanupDrainsBacklog(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// 250 overdue entries exceed the default batch size of 128, forcing
		// the worker through truncated passes; it must resume on its own.
		const count = 250
		c := New[int, int]()
		defer c.Close()
		for key := range count {
			if err := c.Set(key, key, time.Minute); err != nil {
				t.Fatalf("Set(%d): %v", key, err)
			}
		}
		synctest.Wait()
		time.Sleep(time.Minute - time.Nanosecond)
		synctest.Wait()
		schedulerAssertKeysInt(t, c, count)

		time.Sleep(time.Nanosecond)
		synctest.Wait()
		schedulerAssertKeysInt(t, c, 0)
	})
}

// schedulerAssertKeysInt checks the total entry count of an int-keyed cache;
// schedulerAssertKeys needs concrete keys.
func schedulerAssertKeysInt(t *testing.T, c *Cache[int, int], want int) {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	if got := itemCount(c); got != want || len(c.expirations) != want {
		t.Fatalf("items = %d, expirations = %d, want %d each", got, len(c.expirations), want)
	}
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
	c.mu.Lock()
	defer c.mu.Unlock()
	if itemCount(c) != len(keys) || len(c.expirations) != len(keys) {
		t.Fatalf("items = %d, expirations = %d, want keys %v", itemCount(c), len(c.expirations), keys)
	}
	for _, key := range keys {
		if _, ok := c.items.Load(key); !ok {
			t.Fatalf("key %q missing from physical storage; want keys %v", key, keys)
		}
	}
	for index, item := range c.expirations {
		if item.index != index || storedEntry(c, item.key) != item {
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

func TestSchedulerEqualDeadlineUpdatesNeedNoWake(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := manualCache[string, int](t)
		schedulerSet(t, c, "first", 1, time.Minute)
		<-c.wake
		// Virtual time does not advance: both inserts and the overwrite keep
		// the exact same deadline, even though heap-root identity may change.
		schedulerSet(t, c, "second", 2, time.Minute)
		schedulerSet(t, c, "first", 3, time.Minute)
		c.Delete("first")
		select {
		case <-c.wake:
			t.Fatal("same deadline unexpectedly rescheduled timer")
		default:
		}
		schedulerAssertGet(t, c, "second", 2, true)
		c.Delete("second")
		select {
		case <-c.wake:
		default:
			t.Fatal("empty heap did not notify timer shutdown")
		}
	})
}

func TestCapacityReclaimsAtExactDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := manualCache[int, int](t)
		WithMaxEntries[int, int](1)(c)
		if err := c.Set(1, 1, time.Minute); err != nil {
			t.Fatal(err)
		}
		old := storedEntry(c, 1)
		<-c.wake
		time.Sleep(time.Minute)
		if !time.Now().Equal(old.deadline) {
			t.Fatal("test did not reach the exact deadline")
		}
		if err := c.Set(2, 2, time.Hour); err != nil {
			t.Fatalf("Set at the old entry's deadline = %v", err)
		}
		if storedEntry(c, 1) != nil || old.index != -1 || len(c.expirations) != 1 {
			t.Fatal("Set did not replace the expired entry")
		}
		select {
		case <-c.wake:
		default:
			t.Fatal("replacement deadline did not notify the worker")
		}
	})
}

func TestSchedulerCapacityDeleteAndReinsert(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New[string, int](WithMaxEntries[string, int](1))
		defer c.Close()
		schedulerSet(t, c, "first", 1, time.Hour)
		synctest.Wait()
		c.Delete("first")
		schedulerSet(t, c, "second", 2, time.Minute)
		synctest.Wait()
		time.Sleep(time.Minute)
		synctest.Wait()
		schedulerAssertKeys(t, c)
		schedulerSet(t, c, "third", 3, time.Hour)
		synctest.Wait()
		schedulerAssertKeys(t, c, "third")
	})
}
