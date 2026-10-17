package cache

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestCacheConcurrentCapacityProtectionStress keeps live sentinels while a
// bounded cache repeatedly rejects, deletes, expires and admits distinct keys.
// Exact admission counts detect both silent live eviction and lack of progress.
func TestCacheConcurrentCapacityProtectionStress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping capacity protection stress in short mode")
	}
	for _, tc := range []struct {
		name string
		opts []Option[int, int]
	}{
		{name: "default"},
		{name: "one_per_pass", opts: []Option[int, int]{WithCleanupBatchSize[int, int](1), WithCleanupBatchBudget[int, int](0)}},
		{name: "tiny_budget", opts: []Option[int, int]{WithCleanupBatchSize[int, int](0), WithCleanupBatchBudget[int, int](time.Nanosecond)}},
		{name: "unlimited_pass", opts: []Option[int, int]{WithCleanupBatchSize[int, int](0), WithCleanupBatchBudget[int, int](0)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const limit, sentinels, workers, rounds = 32, 8, 128, 8
			c := New[int, int](append(tc.opts, WithMaxEntries[int, int](limit))...)
			t.Cleanup(c.Close)
			for key := 0; key < limit; key++ {
				ttl := 2 * time.Hour
				if key < sentinels {
					ttl = time.Hour // Oldest live deadlines must never be evicted.
				}
				if err := c.Set(key, key, ttl); err != nil {
					t.Fatal(err)
				}
			}
			active := make([]int, 0, limit-sentinels)
			for key := sentinels; key < limit; key++ {
				active = append(active, key)
			}
			for round := 0; round < rounds; round++ {
				base := limit + round*workers*2
				if admitted := capacityAdmissionWave(t, c, base, workers, sentinels); len(admitted) != 0 {
					t.Fatalf("full live cache admitted %d new keys", len(admitted))
				}
				for _, key := range active {
					if round%2 == 0 {
						c.Delete(key)
					} else if err := c.Set(key, key, time.Nanosecond); err != nil {
						t.Fatal(err)
					}
				}
				// All refreshed deadlines are past before the next admission wave.
				// The real cleaner may reclaim them first; Set may also reclaim them.
				time.Sleep(time.Nanosecond)
				active = capacityAdmissionWave(t, c, base+workers, workers, sentinels)
				if len(active) != limit-sentinels {
					t.Fatalf("round %d admitted %d keys after releasing slots, want %d", round, len(active), limit-sentinels)
				}
				assertCapacityConsistent(t, c)
			}
		})
	}
}

func capacityAdmissionWave(t *testing.T, c *Cache[int, int], base, workers, sentinels int) []int {
	t.Helper()
	start := make(chan struct{})
	admitted := make(chan int, workers)
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(key int) {
			defer wg.Done()
			<-start
			if err := c.Set(key, key, 2*time.Hour); err == nil {
				admitted <- key
			} else if !errors.Is(err, ErrCapacity) {
				t.Errorf("Set(%d) = %v", key, err)
			}
			for sentinel := 0; sentinel < sentinels; sentinel++ {
				if got, ok := c.Get(sentinel); !ok || got != sentinel {
					t.Errorf("live sentinel %d = (%d, %t)", sentinel, got, ok)
					return
				}
			}
		}(base + worker)
	}
	close(start)
	wg.Wait()
	close(admitted)
	var keys []int
	for key := range admitted {
		keys = append(keys, key)
		if got, ok := c.Get(key); !ok || got != key {
			t.Errorf("successful admission %d = (%d, %t)", key, got, ok)
		}
	}
	return keys
}

// TestCacheConcurrentHotKeyStress distinguishes generations of one hot key.
// Reads overlapping writes may see any writer in that wave. Barrier-separated
// reads must see the completed marker, deletion or reinsertion exactly.
func TestCacheConcurrentHotKeyStress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping hot-key stress in short mode")
	}
	const workers, rounds = 32, 32
	c := New[int, int]()
	t.Cleanup(c.Close)
	for round := 1; round <= rounds; round++ {
		base := round * (workers + 1)
		if err := c.Set(0, -round, time.Nanosecond); err != nil {
			t.Fatal(err)
		}
		concurrentCacheWave(workers, func(worker int) {
			if err := c.Set(0, base+worker, time.Hour); err != nil {
				t.Error(err)
				return
			}
			if got, ok := c.Get(0); !ok || got < base || got >= base+workers {
				t.Errorf("overlapping Get = (%d, %t), want generation %d", got, ok, round)
			}
			// Keep cleanup active alongside refreshes without deleting the hot key.
			if err := c.Set(worker+1, worker, time.Nanosecond); err != nil {
				t.Error(err)
			}
		})
		marker := base + workers
		if err := c.Set(0, marker, time.Hour); err != nil {
			t.Fatal(err)
		}
		concurrentCacheWave(workers, func(_ int) {
			if got, ok := c.Get(0); !ok || got != marker {
				t.Errorf("Get after completed refresh = (%d, %t), want %d", got, ok, marker)
			}
		})
		concurrentCacheWave(workers, func(_ int) { c.Delete(0) })
		concurrentCacheWave(workers, func(_ int) {
			if got, ok := c.Get(0); ok || got != 0 {
				t.Errorf("Get after completed deletion = (%d, %t)", got, ok)
			}
		})
		if err := c.Set(0, -marker, time.Hour); err != nil {
			t.Fatal(err)
		}
		concurrentCacheWave(workers, func(_ int) {
			if got, ok := c.Get(0); !ok || got != -marker {
				t.Errorf("Get after reinsertion = (%d, %t), want %d", got, ok, -marker)
			}
		})
	}
}

func concurrentCacheWave(workers int, operation func(int)) {
	start := make(chan struct{})
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			<-start
			operation(worker)
		}(worker)
	}
	close(start)
	wg.Wait()
}

// Repeated construction must not leave a cleaner behind, including when work
// and Close overlap. Check the owned done channel rather than global goroutine
// counts, which include unrelated runtime and test goroutines.
func TestRepeatedConcurrentCacheLifecycle(t *testing.T) {
	for iteration := 0; iteration < 32; iteration++ {
		t.Run(fmt.Sprint(iteration), func(t *testing.T) {
			c := New[int, int]()
			t.Cleanup(c.Close)
			concurrentCacheWave(16, func(worker int) {
				if worker%4 == 0 {
					c.Close()
					return
				}
				if err := c.Set(worker, worker, time.Nanosecond); err != nil && !errors.Is(err, ErrClosed) {
					t.Error(err)
				}
				c.Get(worker)
				c.Delete(worker)
			})
			c.Close()
			select {
			case <-c.done:
			default:
				t.Fatal("Close left the cleaner running")
			}
			if itemCount(c) != 0 || len(c.expirations) != 0 {
				t.Fatal("Close retained cache entries")
			}
		})
	}
}
