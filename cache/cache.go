// Package cache provides a concurrent, in-memory cache with per-entry TTLs.
//
// Entries expire at their deadline even if the background cleaner has not yet
// removed them. Construct a Cache with New and call Close when it is no longer
// needed to release its cleaner goroutine and stored values.
package cache

import (
	"container/heap"
	"errors"
	"sync"
	"time"
)

var (
	// ErrClosed is returned by Set after the cache has been closed.
	ErrClosed = errors.New("cache: closed")
	// ErrInvalidTTL is returned by Set for a nonpositive TTL.
	ErrInvalidTTL = errors.New("cache: TTL must be positive")
	// ErrInvalidKey is returned by Set for a key that is not equal to itself,
	// such as a floating-point NaN or a comparable struct containing one.
	ErrInvalidKey = errors.New("cache: key must be equal to itself")
)

// Cache stores values by key until their individual expiration deadlines.
// All methods are safe for concurrent use. Values are not copied: callers must
// synchronize access to mutable data referenced by a stored value themselves.
// Keys follow Go map comparability rules; Set additionally rejects keys that
// are not equal to themselves, such as floating-point NaNs.
//
// A Cache must be created with New and must not be copied. Its zero value is
// not usable. There is no capacity limit or eviction policy other than TTL.
type Cache[K comparable, V any] struct {
	// heapMu serializes all writers, cleanup, and shutdown. Get uses only
	// sync.Map and immutable key/value/deadline fields, never entry indexes.
	heapMu      sync.Mutex
	items       sync.Map // K -> *entry[K, V]
	expirations expiryHeap[K, V]
	closed      bool
	wake        chan struct{}
	stop        chan struct{}
	done        chan struct{}
}

// New creates an empty cache and starts its background expiration cleaner.
// The caller must eventually call Close.
func New[K comparable, V any]() *Cache[K, V] {
	c := &Cache[K, V]{
		wake: make(chan struct{}, 1),
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
	go c.run()
	return c
}

// Set stores value with a new deadline ttl from now, replacing any previous
// value and deadline for key. TTL must be positive; otherwise ErrInvalidTTL is
// returned without changing an existing entry. Keys that are not equal to
// themselves (such as NaN) return ErrInvalidKey without storing the value.
// A closed cache always returns ErrClosed. Updating a key does not create
// another retained expiration record. It replaces the old node in place in
// the heap; published values and deadlines are never mutated.
func (c *Cache[K, V]) Set(key K, value V, ttl time.Duration) error {
	c.heapMu.Lock()
	defer c.heapMu.Unlock()
	if c.closed {
		return ErrClosed
	}
	if ttl <= 0 {
		return ErrInvalidTTL
	}
	if key != key { //nolint:gocritic // NaN-containing keys cannot be retrieved or deleted.
		return ErrInvalidKey
	}
	before, scheduled := c.earliest()
	item := &entry[K, V]{key: key, value: value, deadline: time.Now().Add(ttl)}
	if previous, exists := c.items.Load(key); exists {
		// Writers share heapMu, so the loaded node still owns this index.
		old := previous.(*entry[K, V])
		item.index = old.index
		c.expirations[item.index] = item
		old.index = -1
		heap.Fix(&c.expirations, item.index)
	} else {
		heap.Push(&c.expirations, item)
	}
	// Publish only after repairing the heap. Readers holding the previous
	// node can safely finish without observing a partially updated deadline.
	c.items.Store(key, item)
	c.notifyIfChanged(before, scheduled)
	return nil
}

// Get returns the value for key if it exists and has not expired. At or after
// its deadline, a key is a miss even before physical removal by the cleaner.
// Get does not refresh the deadline. A miss or closed cache returns V's zero
// value and false.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	if stored, ok := c.items.Load(key); ok {
		item := stored.(*entry[K, V])
		if time.Now().Before(item.deadline) {
			return item.value, true
		}
	}
	var zero V
	return zero, false
}

// Delete removes key and its expiration record. It is a no-op if the key is
// missing or the cache is closed.
func (c *Cache[K, V]) Delete(key K) {
	c.heapMu.Lock()
	defer c.heapMu.Unlock()
	before, scheduled := c.earliest()
	if stored, ok := c.items.Load(key); ok {
		heap.Remove(&c.expirations, stored.(*entry[K, V]).index)
		c.items.Delete(key)
		c.notifyIfChanged(before, scheduled)
	}
}

// Close clears the cache and waits for its cleaner to stop. It is safe to call
// repeatedly or concurrently. Once closed, Get always misses, Set returns
// ErrClosed, and Delete is a no-op.
func (c *Cache[K, V]) Close() {
	c.heapMu.Lock()
	if !c.closed {
		c.closed = true
		c.items.Clear()
		c.expirations = nil
		close(c.stop)
	}
	c.heapMu.Unlock()
	<-c.done
}

// earliest returns the next deadline while heapMu is held.
func (c *Cache[K, V]) earliest() (time.Time, bool) {
	if len(c.expirations) == 0 {
		return time.Time{}, false
	}
	return c.expirations[0].deadline, true
}

// notifyIfChanged coalesces changes to the earliest deadline, including a
// transition to or from an empty heap. Callers hold heapMu throughout mutation
// and comparison; the cleaner rechecks the heap under that lock after waking.
func (c *Cache[K, V]) notifyIfChanged(before time.Time, scheduled bool) {
	after, nowScheduled := c.earliest()
	if scheduled == nowScheduled && (!scheduled || before.Equal(after)) {
		return
	}
	select {
	case c.wake <- struct{}{}:
	default:
	}
}

func (c *Cache[K, V]) run() {
	defer close(c.done)
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		delay, scheduled := c.nextDelay()
		if scheduled {
			// Go 1.24+ Reset discards notifications from the previous deadline.
			timer.Reset(delay)
		} else {
			timer.Stop()
		}
		select {
		case <-c.stop:
			return
		case <-c.wake:
		case <-timer.C:
		}
	}
}

// nextDelay removes due entries and returns the wait for the earliest remaining
// deadline. No key has more than one heap node, including after refreshes.
func (c *Cache[K, V]) nextDelay() (time.Duration, bool) {
	c.heapMu.Lock()
	defer c.heapMu.Unlock()
	for len(c.expirations) > 0 {
		item := c.expirations[0]
		if delay := time.Until(item.deadline); delay > 0 {
			return delay, true
		}
		heap.Pop(&c.expirations)
		c.items.Delete(item.key)
	}
	return 0, false
}
