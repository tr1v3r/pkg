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
	// Writers always acquire heapMu before mapMu. Get only needs mapMu.
	// heapMu protects expirations, entry indexes, and closed; mapMu protects
	// items and entry values. Deadline writes require both locks because
	// Get reads them under mapMu and heap operations read them under heapMu.
	heapMu      sync.Mutex
	mapMu       sync.RWMutex
	items       map[K]*entry[K, V]
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
		items: make(map[K]*entry[K, V]),
		wake:  make(chan struct{}, 1),
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}
	go c.run()
	return c
}

// Set stores value with a new deadline ttl from now, replacing any previous
// value and deadline for key. TTL must be positive; otherwise ErrInvalidTTL is
// returned without changing an existing entry. Keys that are not equal to
// themselves (such as NaN) return ErrInvalidKey without storing the value.
// A closed cache always returns ErrClosed. Updating a key does not create
// another expiration record.
func (c *Cache[K, V]) Set(key K, value V, ttl time.Duration) error {
	c.heapMu.Lock()
	defer c.heapMu.Unlock()
	if c.closed {
		return ErrClosed
	}
	if ttl <= 0 {
		return ErrInvalidTTL
	}
	if key != key { //nolint:gocritic // NaN-containing keys cannot be retrieved or deleted from a Go map.
		return ErrInvalidKey
	}

	before, scheduled := c.earliest()
	item, exists := c.store(key, value, time.Now().Add(ttl))
	if exists {
		heap.Fix(&c.expirations, item.index)
	} else {
		heap.Push(&c.expirations, item)
	}
	c.notifyIfChanged(before, scheduled)
	return nil
}

// store publishes a value and deadline while heapMu is held. The map lock is
// released before heap maintenance, so readers do not wait for heap adjustments.
func (c *Cache[K, V]) store(key K, value V, deadline time.Time) (*entry[K, V], bool) {
	c.mapMu.Lock()
	defer c.mapMu.Unlock()
	if item, ok := c.items[key]; ok {
		item.value = value
		item.deadline = deadline
		return item, true
	}
	item := &entry[K, V]{key: key, value: value, deadline: deadline}
	c.items[key] = item
	return item, false
}

// Get returns the value for key if it exists and has not expired. At or after
// its deadline, a key is a miss even before physical removal by the cleaner.
// Get does not refresh the deadline. A miss or closed cache returns V's zero
// value and false.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mapMu.RLock()
	defer c.mapMu.RUnlock()
	if item, ok := c.items[key]; ok && time.Now().Before(item.deadline) {
		return item.value, true
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
	if item := c.remove(key); item != nil {
		heap.Remove(&c.expirations, item.index)
		c.notifyIfChanged(before, scheduled)
	}
}

// remove deletes a map entry while heapMu is held. Heap maintenance is kept
// outside mapMu; heapMu prevents refreshes or reinserts during either step.
func (c *Cache[K, V]) remove(key K) *entry[K, V] {
	c.mapMu.Lock()
	defer c.mapMu.Unlock()
	item := c.items[key]
	delete(c.items, key)
	return item
}

// Close clears the cache and waits for its cleaner to stop. It is safe to call
// repeatedly or concurrently. Once closed, Get always misses, Set returns
// ErrClosed, and Delete is a no-op.
func (c *Cache[K, V]) Close() {
	c.heapMu.Lock()
	if !c.closed {
		c.closed = true
		c.mapMu.Lock()
		c.items = nil
		c.mapMu.Unlock()
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
		c.remove(item.key)
	}
	return 0, false
}
