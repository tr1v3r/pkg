package cache

import (
	"container/heap"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMaxEntriesOptions(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts []Option[int, int]
		want int
	}{
		{name: "default"},
		{name: "zero", opts: []Option[int, int]{WithMaxEntries[int, int](0)}},
		{name: "negative", opts: []Option[int, int]{WithMaxEntries[int, int](-1)}, want: -1},
		{name: "positive", opts: []Option[int, int]{WithMaxEntries[int, int](2)}, want: 2},
		{name: "last wins", opts: []Option[int, int]{WithMaxEntries[int, int](1), WithMaxEntries[int, int](2)}, want: 2},
		{name: "disable", opts: []Option[int, int]{WithMaxEntries[int, int](1), WithMaxEntries[int, int](0)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := New[int, int](tc.opts...)
			t.Cleanup(c.Close)
			if c.maxEntries != tc.want {
				t.Fatalf("maxEntries = %d, want %d", c.maxEntries, tc.want)
			}
			for key := 0; key < 3; key++ {
				err := c.Set(key, key, time.Hour)
				if tc.want > 0 && key >= tc.want {
					if !errors.Is(err, ErrCapacity) {
						t.Fatalf("Set(%d) = %v, want ErrCapacity", key, err)
					}
				} else if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestCapacityRejectsWithoutMutation(t *testing.T) {
	c := manualCache[float64, int](t)
	WithMaxEntries[float64, int](1)(c)
	if err := c.Set(1, 42, time.Hour); err != nil {
		t.Fatal(err)
	}
	before := *storedEntry(c, 1)
	<-c.wake
	for _, tc := range []struct {
		name string
		key  float64
		ttl  time.Duration
		want error
	}{
		{name: "full", key: 2, ttl: time.Hour, want: ErrCapacity},
		{name: "TTL before capacity", key: 2, ttl: 0, want: ErrInvalidTTL},
		{name: "key before capacity", key: math.NaN(), ttl: time.Hour, want: ErrInvalidKey},
		{name: "TTL before key", key: math.NaN(), ttl: 0, want: ErrInvalidTTL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := c.Set(tc.key, 99, tc.ttl); !errors.Is(err, tc.want) {
				t.Fatalf("Set = %v, want %v", err, tc.want)
			}
			if itemCount(c) != 1 || len(c.expirations) != 1 || *storedEntry(c, 1) != before {
				t.Fatal("rejected Set changed the stored entry or heap")
			}
			if c.expirations[0] != storedEntry(c, 1) || len(c.wake) != 0 {
				t.Fatal("rejected Set changed the heap or notified the worker")
			}
		})
	}
	c.Close()
	if err := c.Set(math.NaN(), 0, 0); !errors.Is(err, ErrClosed) {
		t.Fatalf("Set after Close = %v, want ErrClosed", err)
	}
}

func TestCapacityRefreshAndDelete(t *testing.T) {
	for _, expired := range []bool{false, true} {
		t.Run(fmt.Sprintf("expired=%t", expired), func(t *testing.T) {
			c := manualCache[int, string](t)
			WithMaxEntries[int, string](1)(c)
			if err := c.Set(1, "old", time.Hour); err != nil {
				t.Fatal(err)
			}
			if expired {
				expireAll(c)
			}
			old := storedEntry(c, 1)
			deadline := old.deadline
			if err := c.Set(1, "a larger replacement value", 2*time.Hour); err != nil {
				t.Fatal(err)
			}
			if got, ok := c.Get(1); !ok || got != "a larger replacement value" {
				t.Fatalf("Get after refresh = (%q, %t)", got, ok)
			}
			if len(c.expirations) != 1 || itemCount(c) != 1 || old.index != -1 ||
				old.value != "old" || !old.deadline.Equal(deadline) {
				t.Fatal("refresh changed the old snapshot or added a retained node")
			}
			if err := c.Set(2, "new", time.Hour); !errors.Is(err, ErrCapacity) {
				t.Fatalf("Set at capacity = %v, want ErrCapacity", err)
			}
			c.Delete(1)
			if err := c.Set(2, "new", time.Hour); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCapacityReclaimsOnlyOneExpiredEntry(t *testing.T) {
	for _, backlog := range []int{1, 2, 10000} {
		t.Run(fmt.Sprint(backlog), func(t *testing.T) {
			c := manualCache[int, int](t)
			WithMaxEntries[int, int](backlog + 1)(c)
			// One live entry remains protected even with an overdue backlog.
			if err := c.Set(-1, -1, time.Hour); err != nil {
				t.Fatal(err)
			}
			deadline := time.Now().Add(-time.Hour)
			for key := 0; key < backlog; key++ {
				item := &entry[int, int]{key: key, value: key, deadline: deadline}
				heap.Push(&c.expirations, item)
				c.items.Store(key, item)
			}
			root := c.expirations[0]
			live := storedEntry(c, -1)
			<-c.wake
			if err := c.Set(-2, -2, 2*time.Hour); err != nil {
				t.Fatal(err)
			}
			if storedEntry(c, root.key) != nil || root.index != -1 {
				t.Fatal("expired root was not removed from map and heap")
			}
			if len(c.expirations) != backlog+1 || itemCount(c) != backlog+1 || storedEntry(c, -1) != live {
				t.Fatal("Set reclaimed more than one entry or removed the live entry")
			}
			if got, ok := c.Get(-2); !ok || got != -2 {
				t.Fatal("new entry was not published")
			}
			// Equal overdue deadlines leave the worker's schedule unchanged.
			if wantWake := backlog == 1; (len(c.wake) != 0) != wantWake {
				t.Fatalf("worker notification = %t, want %t", len(c.wake) != 0, wantWake)
			}
			assertCapacityConsistent(t, c)
		})
	}
}

func TestCapacityConcurrentAdmission(t *testing.T) {
	const limit, workers = 32, 128
	c := New[int, int](WithMaxEntries[int, int](limit))
	t.Cleanup(c.Close)
	var admitted atomic.Int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for key := 0; key < workers; key++ {
		wg.Add(1)
		go func(key int) {
			defer wg.Done()
			<-start
			err := c.Set(key, key, time.Hour)
			if err == nil {
				admitted.Add(1)
			} else if !errors.Is(err, ErrCapacity) {
				t.Errorf("Set = %v", err)
			}
		}(key)
	}
	close(start)
	wg.Wait()
	if got := admitted.Load(); got != limit {
		t.Fatalf("admitted %d distinct keys, want %d", got, limit)
	}
	assertCapacityConsistent(t, c)
}

// assertCapacityConsistent checks a stable view while excluding every writer.
func assertCapacityConsistent(t *testing.T, c *Cache[int, int]) {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	if count := itemCount(c); count != len(c.expirations) || count > c.maxEntries {
		t.Fatalf("map=%d heap=%d limit=%d", count, len(c.expirations), c.maxEntries)
	}
	for index, item := range c.expirations {
		if item.index != index || storedEntry(c, item.key) != item {
			t.Fatalf("broken map/heap index at %d", index)
		}
		if index > 0 && c.expirations.Less(index, (index-1)/2) {
			t.Fatalf("broken heap ordering at %d", index)
		}
	}
}

// Use a manual cache to exclude worker scheduling. Each measured Set sees a
// full heap with an expired root. Restore overdue deadlines once per batch,
// outside the measured region, to amortize benchmark timer bookkeeping.
func BenchmarkCacheCapacityReclaim(b *testing.B) {
	for _, entries := range []int{1024, 65536} {
		b.Run(fmt.Sprint(entries), func(b *testing.B) {
			c := &Cache[int, int]{
				wake:       make(chan struct{}, 1),
				maxEntries: entries,
			}
			deadline := time.Now().Add(-time.Hour)
			for key := 0; key < entries; key++ {
				item := &entry[int, int]{key: key, value: key, deadline: deadline}
				heap.Push(&c.expirations, item)
				c.items.Store(key, item)
			}
			key := entries
			b.ReportAllocs()
			b.ResetTimer()
			for remaining := b.N; remaining > 0; {
				batch := min(remaining, entries)
				for range batch {
					if err := c.Set(key, key, time.Hour); err != nil {
						b.Fatal(err)
					}
					key++
				}
				remaining -= batch
				if remaining > 0 {
					b.StopTimer()
					// No reader or cleaner observes these benchmark-only nodes.
					expireAll(c)
					b.StartTimer()
				}
			}
		})
	}
}
