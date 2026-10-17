package cache

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

const (
	cacheStressWorkers = 128
	cacheStressRounds  = 128
	cacheStressBacklog = 32768
	cacheStressClosers = 16
)

// TestCacheConcurrentCleanupStress uses the real worker and default cleanup
// limits. It checks correctness under contention, not a latency or QPS target.
func TestCacheConcurrentCleanupStress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cache cleanup stress test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	c := New[int, int]()
	start := make(chan struct{})
	closing := make(chan struct{})
	errs := make(chan error, cacheStressWorkers+cacheStressClosers)
	var ready, warm, all sync.WaitGroup
	ready.Add(cacheStressWorkers + cacheStressClosers)
	warm.Add(cacheStressWorkers)
	t.Cleanup(func() {
		cancel() // Release workers waiting at either barrier on assertion failures.
		c.Close()
		all.Wait()
	})

	// Hold the existing heap mutex until every goroutine is ready. The real
	// cleaner and writers then contend for the same cache, without test hooks
	// or any assumption about which goroutine acquires the mutex first.
	c.mu.Lock()
	deadline := seedCacheStressBacklog(c)
	for worker := 0; worker < cacheStressWorkers; worker++ {
		all.Add(1)
		go func(worker int) {
			defer all.Done()
			ready.Done()
			select {
			case <-start:
			case <-ctx.Done():
				warm.Done()
				return
			}
			err := runCacheStressOperations(ctx, c, worker, false)
			if err != nil {
				errs <- err
			}
			warm.Done()
			if err != nil {
				return
			}
			select {
			case <-closing:
			case <-ctx.Done():
				return
			}
			if err := runCacheStressOperations(ctx, c, worker, true); err != nil {
				errs <- err
			}
		}(worker)
	}
	for closer := 0; closer < cacheStressClosers; closer++ {
		all.Add(1)
		go func() {
			defer all.Done()
			ready.Done()
			select {
			case <-closing:
			case <-ctx.Done():
				return
			}
			c.Close()
			if err := c.Set(-1, 0, time.Hour); !errors.Is(err, ErrClosed) {
				errs <- fmt.Errorf("Set after Close = %v, want ErrClosed", err)
				return
			}
			if _, ok := c.Get(-1); ok {
				errs <- errors.New("Get after Close returned a value")
				return
			}
			select {
			case <-c.done:
			default:
				errs <- errors.New("Close returned before the real worker exited")
			}
		}()
	}
	ready.Wait()
	close(start)
	c.mu.Unlock()

	waitCacheStressGroup(t, ctx, &warm)
	select {
	case err := <-errs:
		t.Fatal(err)
	default:
	}
	waitCacheStressCleanup(t, ctx, c, deadline)
	assertCacheStressConsistency(t, c)

	// Refill the overdue backlog before releasing the second barrier. Close
	// races with the real cleaner and another round of mixed operations.
	// It is valid for Close or cleanup to win; neither ordering is required.
	c.mu.Lock()
	seedCacheStressBacklog(c)
	close(closing)
	c.mu.Unlock()
	waitCacheStressGroup(t, ctx, &all)
	close(errs)
	for err := range errs {
		t.Error(err)
	}

	assertCacheStressConsistency(t, c)
	c.mu.Lock()
	closed, items, expirations := c.closed, itemCount(c), len(c.expirations)
	c.mu.Unlock()
	if !closed || items != 0 || expirations != 0 {
		t.Fatalf("after concurrent Close: closed = %t, items = %d, expirations = %d", closed, items, expirations)
	}
	select {
	case <-c.done:
	default:
		t.Fatal("real cleanup worker did not exit")
	}
}

// seedCacheStressBacklog publishes already-overdue snapshots without mutating
// any published deadline. The caller holds c.mu and previous backlog keys have
// been reclaimed before they are reused.
func seedCacheStressBacklog(c *Cache[int, int]) time.Time {
	before, scheduled := c.earliest()
	deadline := time.Now().Add(-time.Hour)
	for key := 0; key < cacheStressBacklog; key++ {
		item := &entry[int, int]{key: key, value: key, deadline: deadline}
		heap.Push(&c.expirations, item)
		c.items.Store(key, item)
	}
	c.notifyIfChanged(before, scheduled)
	return deadline
}

func runCacheStressOperations(ctx context.Context, c *Cache[int, int], worker int, closing bool) error {
	// Private keys permit exact read-after-write checks. Shared keys exercise
	// concurrent refresh/delete/reinsertion; their values equal their keys.
	privateKey := -worker - 1
	for step := 0; step < cacheStressRounds; step++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		ttl := time.Hour
		if step%2 != 0 {
			ttl = time.Nanosecond
		}
		value := worker*cacheStressRounds + step
		err := c.Set(privateKey, value, ttl)
		if err != nil && (!closing || !errors.Is(err, ErrClosed)) {
			return fmt.Errorf("worker %d private Set: %w", worker, err)
		}
		got, ok := c.Get(privateKey)
		if ok && got != value {
			return fmt.Errorf("worker %d Get = %d, want %d", worker, got, value)
		}
		if !closing && ttl == time.Hour && !ok {
			return fmt.Errorf("worker %d lost a live private key", worker)
		}
		if step%3 == 0 {
			c.Delete(privateKey)
			if _, ok := c.Get(privateKey); ok {
				return fmt.Errorf("worker %d Delete left a readable private key", worker)
			}
		}
		if _, ok := c.Get(step % cacheStressBacklog); ok {
			return errors.New("overdue backlog key was readable")
		}

		sharedKey := -cacheStressWorkers - 1 - step%8
		err = c.Set(sharedKey, sharedKey, ttl)
		if err != nil && (!closing || !errors.Is(err, ErrClosed)) {
			return fmt.Errorf("worker %d shared Set: %w", worker, err)
		}
		if got, ok := c.Get(sharedKey); ok && got != sharedKey {
			return fmt.Errorf("shared Get(%d) = %d", sharedKey, got)
		}
		if step%4 == 0 {
			c.Delete(sharedKey)
		}
	}
	return nil
}

func waitCacheStressGroup(t *testing.T, ctx context.Context, group *sync.WaitGroup) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		group.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatalf("concurrent cache operations did not finish: %v", ctx.Err())
	}
}

func waitCacheStressCleanup(t *testing.T, ctx context.Context, c *Cache[int, int], deadline time.Time) {
	t.Helper()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		c.mu.Lock()
		due := len(c.expirations) > 0 && !c.expirations[0].deadline.After(deadline)
		c.mu.Unlock()
		if !due {
			return
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatal("real worker did not reclaim the overdue backlog")
		}
	}
}

func assertCacheStressConsistency(t *testing.T, c *Cache[int, int]) {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	if count := itemCount(c); count != len(c.expirations) {
		t.Fatalf("map has %d entries, heap has %d", count, len(c.expirations))
	}
	for index, item := range c.expirations {
		if item.key >= 0 {
			t.Fatalf("overdue backlog key %d was not reclaimed", item.key)
		}
		if item.index != index || storedEntry(c, item.key) != item {
			t.Fatalf("heap node %d is not indexed consistently in the map and heap", index)
		}
		if index > 0 && c.expirations.Less(index, (index-1)/2) {
			t.Fatalf("heap node %d violates minimum-heap ordering", index)
		}
	}
}

// TestCacheConcurrentCapacityStress exercises admission alongside the real
// cleaner, deletes, refreshes and shutdown. It does not measure capacity in QPS.
func TestCacheConcurrentCapacityStress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cache capacity stress test in short mode")
	}
	const limit, workers, rounds = 64, 128, 256
	c := New[int, int](WithMaxEntries[int, int](limit))
	t.Cleanup(c.Close)

	// A complete workload runs before the shutdown race so an early Close
	// cannot turn all admission and expiration checks into closed-cache checks.
	for phase := 0; phase < 2; phase++ {
		var ready, all sync.WaitGroup
		start := make(chan struct{})
		for worker := 0; worker < workers; worker++ {
			ready.Add(1)
			all.Add(1)
			go func(worker int) {
				defer all.Done()
				ready.Done()
				<-start
				for round := 0; round < rounds; round++ {
					key := (worker + round*17) % (limit * 4)
					ttl := time.Hour
					if round%2 == 0 {
						ttl = time.Nanosecond
					}
					err := c.Set(key, key, ttl)
					if err != nil && !errors.Is(err, ErrCapacity) &&
						(phase == 0 || !errors.Is(err, ErrClosed)) {
						t.Errorf("Set during phase %d = %v", phase, err)
						return
					}
					if value, ok := c.Get(key); ok && value != key {
						t.Errorf("Get(%d) = %d", key, value)
						return
					}
					if round%3 == 0 {
						c.Delete((key + 1) % (limit * 4))
					}
					if round%16 == 0 {
						assertCapacityConsistent(t, c)
					}
				}
			}(worker)
		}
		if phase == 1 {
			for closer := 0; closer < cacheStressClosers; closer++ {
				ready.Add(1)
				all.Add(1)
				go func() {
					defer all.Done()
					ready.Done()
					<-start
					c.Close()
				}()
			}
		}
		ready.Wait()
		close(start)
		all.Wait()
		assertCapacityConsistent(t, c)
	}
	if err := c.Set(0, 0, time.Hour); !errors.Is(err, ErrClosed) {
		t.Fatalf("Set after Close = %v, want ErrClosed", err)
	}
	if _, ok := c.Get(0); ok {
		t.Fatal("closed cache returned a value")
	}
}
