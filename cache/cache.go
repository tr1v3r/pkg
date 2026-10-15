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
	mu          sync.RWMutex
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
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return ErrClosed
	}
	if ttl <= 0 {
		return ErrInvalidTTL
	}
	if key != key { //nolint:gocritic // NaN-containing keys cannot be retrieved or deleted from a Go map.
		return ErrInvalidKey
	}

	deadline := time.Now().Add(ttl)
	if item, ok := c.items[key]; ok {
		item.value = value
		item.deadline = deadline
		heap.Fix(&c.expirations, item.index)
	} else {
		item := &entry[K, V]{key: key, value: value, deadline: deadline}
		c.items[key] = item
		heap.Push(&c.expirations, item)
	}
	c.notify()
	return nil
}

// Get returns the value for key if it exists and has not expired. At or after
// its deadline, a key is a miss even before physical removal by the cleaner.
// Get does not refresh the deadline. A miss or closed cache returns V's zero
// value and false.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if item, ok := c.items[key]; ok && time.Now().Before(item.deadline) {
		return item.value, true
	}
	var zero V
	return zero, false
}

// Delete removes key and its expiration record. It is a no-op if the key is
// missing or the cache is closed.
func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if item, ok := c.items[key]; ok {
		heap.Remove(&c.expirations, item.index)
		delete(c.items, key)
		c.notify()
	}
}

// Close clears the cache and waits for its cleaner to stop. It is safe to call
// repeatedly or concurrently. Once closed, Get always misses, Set returns
// ErrClosed, and Delete is a no-op.
func (c *Cache[K, V]) Close() {
	c.mu.Lock()
	if !c.closed {
		c.closed = true
		c.items = nil
		c.expirations = nil
		close(c.stop)
	}
	c.mu.Unlock()
	<-c.done
}

// notify coalesces scheduler changes. Callers hold mu while updating state;
// the cleaner always rechecks the heap under that same lock after a wake-up.
func (c *Cache[K, V]) notify() {
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
		var tick <-chan time.Time
		if scheduled {
			// Go 1.24+ Reset discards notifications from the previous deadline.
			timer.Reset(delay)
			tick = timer.C
		} else {
			timer.Stop()
		}
		select {
		case <-c.stop:
			return
		case <-c.wake:
		case <-tick:
		}
	}
}

// nextDelay removes due entries and returns the wait for the earliest remaining
// deadline. No key has more than one heap node, including after refreshes.
func (c *Cache[K, V]) nextDelay() (time.Duration, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for len(c.expirations) > 0 {
		item := c.expirations[0]
		if delay := time.Until(item.deadline); delay > 0 {
			return delay, true
		}
		heap.Pop(&c.expirations)
		delete(c.items, item.key)
	}
	return 0, false
}
