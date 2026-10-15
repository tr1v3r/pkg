package cache

import (
	"container/heap"
	"errors"
	"math"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// A fixture without a cleaner lets tests examine expiration and heap invariants
// independently of scheduling. Lifecycle tests use New instead.
func manualCache[K comparable, V any](t *testing.T) *Cache[K, V] {
	t.Helper()
	c := &Cache[K, V]{
		items: make(map[K]*entry[K, V]),
		wake:  make(chan struct{}, 1),
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}
	close(c.done)
	t.Cleanup(c.Close)
	return c
}

func TestSetGetDelete(t *testing.T) {
	c := New[string, int]()
	t.Cleanup(c.Close)
	if got, ok := c.Get("missing"); got != 0 || ok {
		t.Fatalf("missing key: got (%d, %v)", got, ok)
	}
	for _, value := range []int{0, 42, 99} {
		if err := c.Set("key", value, time.Hour); err != nil {
			t.Fatal(err)
		}
		if got, ok := c.Get("key"); !ok || got != value {
			t.Fatalf("Get: got (%d, %v), want (%d, true)", got, ok, value)
		}
	}
	c.Delete("key")
	c.Delete("missing")
	if got, ok := c.Get("key"); got != 0 || ok {
		t.Fatalf("deleted key: got (%d, %v)", got, ok)
	}
}

func TestGenericKeysAndNilValues(t *testing.T) {
	type key struct {
		group string
		id    int
	}
	c := New[key, *int]()
	t.Cleanup(c.Close)
	k := key{group: "example", id: 1}
	if err := c.Set(k, nil, time.Hour); err != nil {
		t.Fatal(err)
	}
	if got, ok := c.Get(k); got != nil || !ok {
		t.Fatalf("stored nil: got (%v, %v)", got, ok)
	}
}

func assertNonReflexiveKeyRejected[K comparable](t *testing.T, key K) {
	t.Helper()
	c := manualCache[K, *int](t)
	value := 42
	for i := 0; i < 10; i++ {
		if err := c.Set(key, &value, time.Hour); !errors.Is(err, ErrInvalidKey) {
			t.Fatalf("Set error = %v, want ErrInvalidKey", err)
		}
	}
	if len(c.items) != 0 || len(c.expirations) != 0 {
		t.Fatal("rejected key retained a value or expiration node")
	}
	if got, ok := c.Get(key); got != nil || ok {
		t.Fatal("rejected key was readable")
	}
	c.Delete(key)
	if err := c.Set(key, &value, 0); !errors.Is(err, ErrInvalidTTL) {
		t.Fatalf("invalid TTL precedence: %v", err)
	}
	c.Close()
	if err := c.Set(key, &value, time.Hour); !errors.Is(err, ErrClosed) {
		t.Fatalf("closed cache precedence: %v", err)
	}
}

func TestNonReflexiveKeysRejected(t *testing.T) {
	t.Run("float64", func(t *testing.T) { assertNonReflexiveKeyRejected(t, math.NaN()) })
	t.Run("float32", func(t *testing.T) { assertNonReflexiveKeyRejected(t, float32(math.NaN())) })
	t.Run("complex", func(t *testing.T) { assertNonReflexiveKeyRejected(t, complex(1, math.NaN())) })
	t.Run("struct", func(t *testing.T) {
		assertNonReflexiveKeyRejected(t, struct{ F float64 }{F: math.NaN()})
	})
	t.Run("array", func(t *testing.T) { assertNonReflexiveKeyRejected(t, [2]float64{0, math.NaN()}) })
	t.Run("interface", func(t *testing.T) { assertNonReflexiveKeyRejected[any](t, math.NaN()) })
}

func TestFloatKeys(t *testing.T) {
	c := manualCache[float64, int](t)
	for i, key := range []float64{0, -1.5, math.Inf(1), math.Inf(-1)} {
		if err := c.Set(key, i, time.Hour); err != nil {
			t.Fatal(err)
		}
		if got, ok := c.Get(key); !ok || got != i {
			t.Fatalf("Get(%v) = (%d, %v), want %d", key, got, ok, i)
		}
		c.Delete(key)
	}
}

func TestInvalidTTLPreservesEntry(t *testing.T) {
	for _, ttl := range []time.Duration{0, -1, -time.Hour} {
		t.Run(ttl.String(), func(t *testing.T) {
			c := manualCache[string, int](t)
			if err := c.Set("key", 1, time.Hour); err != nil {
				t.Fatal(err)
			}
			deadline := c.items["key"].deadline
			if err := c.Set("key", 2, ttl); !errors.Is(err, ErrInvalidTTL) {
				t.Fatalf("Set error = %v, want ErrInvalidTTL", err)
			}
			if got, ok := c.Get("key"); !ok || got != 1 || !c.items["key"].deadline.Equal(deadline) {
				t.Fatal("invalid Set changed the original value or deadline")
			}
			if err := c.Set("missing", 3, ttl); !errors.Is(err, ErrInvalidTTL) {
				t.Fatalf("missing key Set error = %v", err)
			}
			if len(c.items) != 1 || len(c.expirations) != 1 {
				t.Fatal("invalid Set added a map or heap entry")
			}
		})
	}
}

func TestGetRejectsExpiredBeforeCleanup(t *testing.T) {
	c := manualCache[string, int](t)
	if err := c.Set("key", 7, time.Hour); err != nil {
		t.Fatal(err)
	}
	c.items["key"].deadline = time.Now().Add(-time.Hour)
	if got, ok := c.Get("key"); got != 0 || ok {
		t.Fatalf("expired key: got (%d, %v)", got, ok)
	}
	if len(c.items) != 1 {
		t.Fatal("Get unexpectedly performed physical cleanup")
	}
	if delay, scheduled := c.nextDelay(); scheduled || delay != 0 || len(c.items) != 0 {
		t.Fatal("cleaner failed to remove expired entry")
	}
}

func TestGetDoesNotRefreshTTL(t *testing.T) {
	c := manualCache[string, int](t)
	if err := c.Set("key", 1, time.Hour); err != nil {
		t.Fatal(err)
	}
	deadline := c.items["key"].deadline
	for i := 0; i < 100; i++ {
		c.Get("key")
	}
	if !c.items["key"].deadline.Equal(deadline) {
		t.Fatal("Get refreshed the deadline")
	}
}

func TestHeapRefreshAndDelete(t *testing.T) {
	c := manualCache[string, int](t)
	for i, key := range []string{"late", "early", "middle"} {
		ttl := []time.Duration{3 * time.Hour, time.Hour, 2 * time.Hour}[i]
		if err := c.Set(key, i, ttl); err != nil {
			t.Fatal(err)
		}
	}
	if c.expirations[0].key != "early" {
		t.Fatal("out-of-order inserts did not maintain min-heap")
	}
	if err := c.Set("late", 10, time.Minute); err != nil {
		t.Fatal(err)
	}
	if c.expirations[0].key != "late" {
		t.Fatal("shortened deadline did not move to heap root")
	}
	if err := c.Set("late", 11, 4*time.Hour); err != nil {
		t.Fatal(err)
	}
	if c.expirations[0].key != "early" {
		t.Fatal("extended deadline did not move away from root")
	}
	c.Delete("middle")
	c.Delete("early")
	if len(c.expirations) != 1 || c.expirations[0].key != "late" || c.expirations[0].index != 0 {
		t.Fatal("deletion corrupted the remaining heap index")
	}
	if delay, scheduled := c.nextDelay(); !scheduled || delay <= 0 {
		t.Fatal("future entry did not schedule a positive wait")
	}
}

func TestRefreshDoesNotAccumulateNodes(t *testing.T) {
	c := manualCache[string, int](t)
	for i := 0; i < 10000; i++ {
		if err := c.Set("key", i, time.Hour); err != nil {
			t.Fatal(err)
		}
	}
	if len(c.expirations) != 1 || len(c.items) != 1 {
		t.Fatalf("refresh accumulated nodes: map=%d, heap=%d", len(c.items), len(c.expirations))
	}
	c.Delete("key")
	if len(c.expirations) != 0 || len(c.items) != 0 {
		t.Fatal("Delete retained entries")
	}
	for _, item := range c.expirations[:cap(c.expirations)] {
		if item != nil {
			t.Fatal("heap backing array retained a deleted entry")
		}
	}
}

func TestExpiredOverwriteAndDeleteReinsert(t *testing.T) {
	c := manualCache[string, int](t)
	if err := c.Set("key", 1, time.Hour); err != nil {
		t.Fatal(err)
	}
	c.items["key"].deadline = time.Now().Add(-time.Hour)
	if err := c.Set("key", 2, time.Hour); err != nil {
		t.Fatal(err)
	}
	c.nextDelay()
	if got, ok := c.Get("key"); !ok || got != 2 {
		t.Fatal("old deadline removed the refreshed value")
	}
	c.Delete("key")
	if err := c.Set("key", 3, time.Hour); err != nil {
		t.Fatal(err)
	}
	c.nextDelay()
	if got, ok := c.Get("key"); !ok || got != 3 || len(c.expirations) != 1 {
		t.Fatal("Delete followed by Set did not create one live entry")
	}
}

func TestMaximumTTL(t *testing.T) {
	c := manualCache[int, int](t)
	if err := c.Set(1, 2, time.Duration(1<<63-1)); err != nil {
		t.Fatal(err)
	}
	if value, ok := c.Get(1); !ok || value != 2 {
		t.Fatal("maximum duration expired early")
	}
	if delay, scheduled := c.nextDelay(); !scheduled || delay <= 0 {
		t.Fatal("maximum duration overflowed the scheduled delay")
	}
}

func TestHeapAgainstModel(t *testing.T) {
	c := manualCache[int, int](t)
	model := make(map[int]int)
	rng := rand.New(rand.NewSource(42))
	for step := 0; step < 2000; step++ {
		key := rng.Intn(32)
		switch rng.Intn(4) {
		case 0, 1:
			value := rng.Int()
			if err := c.Set(key, value, time.Duration(1+rng.Intn(100))*time.Hour); err != nil {
				t.Fatal(err)
			}
			model[key] = value
		case 2:
			c.Delete(key)
			delete(model, key)
		case 3:
			if item, ok := c.items[key]; ok {
				item.deadline = time.Now().Add(-time.Hour)
				heap.Fix(&c.expirations, item.index)
			}
			c.nextDelay()
			delete(model, key)
		}
		if len(c.items) != len(model) || len(c.expirations) != len(model) {
			t.Fatalf("step %d: model/map/heap lengths differ", step)
		}
		for index, item := range c.expirations {
			if item.index != index || c.items[item.key] != item {
				t.Fatalf("step %d: broken map/heap index", step)
			}
			if index > 0 && c.expirations.Less(index, (index-1)/2) {
				t.Fatalf("step %d: broken heap ordering", step)
			}
		}
		for key, want := range model {
			if got, ok := c.Get(key); !ok || got != want {
				t.Fatalf("step %d, key %d: got (%d, %v), want %d", step, key, got, ok, want)
			}
		}
	}
}

func TestConcurrentOperationsAndClose(t *testing.T) {
	c := New[int, int]()
	t.Cleanup(c.Close)
	const workers = 16
	start := make(chan struct{})
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			<-start
			for i := 0; i < 300; i++ {
				key := (worker + i) % 32
				if err := c.Set(key, i, time.Hour); err != nil && !errors.Is(err, ErrClosed) {
					errs <- err
					return
				}
				c.Get(key)
				c.Delete((key + 1) % 32)
				if i == 150 {
					c.Close()
				}
			}
		}(worker)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	c.Close()
	if err := c.Set(1, 1, 0); !errors.Is(err, ErrClosed) {
		t.Fatalf("closed Set error = %v, want ErrClosed", err)
	}
	if value, ok := c.Get(1); value != 0 || ok {
		t.Fatal("closed cache returned a value")
	}
	c.heapMu.Lock()
	defer c.heapMu.Unlock()
	if c.items != nil || c.expirations != nil {
		t.Fatal("Close retained stored entries")
	}
	select {
	case <-c.done:
	default:
		t.Fatal("Close returned before cleaner stopped")
	}
}

func TestNotificationsFollowEarliestDeadline(t *testing.T) {
	c := manualCache[string, int](t)
	steps := []struct {
		name     string
		key      string
		ttl      time.Duration
		remove   bool
		wantWake bool
	}{
		{name: "empty to scheduled", key: "first", ttl: time.Hour, wantWake: true},
		{name: "insert later", key: "later", ttl: 2 * time.Hour},
		{name: "refresh later", key: "later", ttl: 3 * time.Hour},
		{name: "shorten to earliest", key: "later", ttl: time.Minute, wantWake: true},
		{name: "extend past earliest", key: "later", ttl: 3 * time.Hour, wantWake: true},
		{name: "delete later", key: "later", remove: true},
		{name: "delete missing", key: "missing", remove: true},
		{name: "insert earlier", key: "earlier", ttl: time.Minute, wantWake: true},
		{name: "delete earliest", key: "earlier", remove: true, wantWake: true},
		{name: "scheduled to empty", key: "first", remove: true, wantWake: true},
	}
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			if step.remove {
				c.Delete(step.key)
			} else if err := c.Set(step.key, 1, step.ttl); err != nil {
				t.Fatal(err)
			}
			select {
			case <-c.wake:
				if !step.wantWake {
					t.Fatal("unchanged earliest deadline notified worker")
				}
			default:
				if step.wantWake {
					t.Fatal("changed earliest deadline did not notify worker")
				}
			}
		})
	}
}

func TestEqualDeadlineRootChangeDoesNotNotify(t *testing.T) {
	c := manualCache[string, int](t)
	deadline := time.Now().Add(time.Hour)
	for index, key := range []string{"first", "second"} {
		item := &entry[string, int]{key: key, deadline: deadline, index: index}
		c.items[key] = item
		c.expirations = append(c.expirations, item)
	}
	c.Delete("first")
	select {
	case <-c.wake:
		t.Fatal("equal deadline needs no timer reschedule")
	default:
	}
	c.Delete("second")
	select {
	case <-c.wake:
	default:
		t.Fatal("empty heap must stop the timer")
	}
}

func TestDeadlineNotificationsCoalesce(t *testing.T) {
	c := manualCache[int, int](t)
	for i := 10; i > 0; i-- {
		if err := c.Set(i, i, time.Duration(i)*time.Hour); err != nil {
			t.Fatal(err)
		}
	}
	if len(c.wake) != 1 {
		t.Fatal("deadline changes did not coalesce into one pending wake")
	}
}

func TestGetDoesNotWaitForHeapLock(t *testing.T) {
	c := manualCache[string, int](t)
	if err := c.Set("live", 42, time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("expired", 7, time.Hour); err != nil {
		t.Fatal(err)
	}
	c.items["expired"].deadline = time.Now().Add(-time.Hour)
	c.heapMu.Lock()
	defer c.heapMu.Unlock()
	result := make(chan bool, 1)
	go func() {
		value, live := c.Get("live")
		_, expired := c.Get("expired")
		_, missing := c.Get("missing")
		result <- value == 42 && live && !expired && !missing
	}()
	select {
	case ok := <-result:
		if !ok {
			t.Fatal("Get returned incorrect visibility while heap was locked")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Get waited for the heap lock")
	}
}

func TestConcurrentRefreshAndCleanup(t *testing.T) {
	c := manualCache[int, int](t)
	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				key := (worker + i) % 64
				ttl := time.Nanosecond
				if i%2 == 0 {
					ttl = time.Hour
				}
				if err := c.Set(key, i, ttl); err != nil {
					t.Error(err)
					return
				}
				c.Get(key)
				if i%3 == 0 {
					c.Delete((key + 1) % 64)
				}
				c.nextDelay()
			}
		}(worker)
	}
	wg.Wait()
	if len(c.items) != len(c.expirations) {
		t.Fatal("concurrent cleanup lost map/heap correspondence")
	}
	for index, item := range c.expirations {
		if item.index != index || c.items[item.key] != item {
			t.Fatal("concurrent refresh corrupted the entry index")
		}
		if index > 0 && c.expirations.Less(index, (index-1)/2) {
			t.Fatal("concurrent refresh corrupted heap ordering")
		}
	}
}
