//go:build !go1.25

package cache

import (
	"testing"
	"time"
)

// Go 1.24 lacks testing/synctest. These smoke tests check eventual physical
// reclamation and timer cancellation with generous watchdogs, not exact timing.
func TestSchedulerRealtime(t *testing.T) {
	t.Run("idle insertion", func(t *testing.T) {
		c := New[int, int]()
		t.Cleanup(c.Close)
		if err := c.Set(1, 1, 20*time.Millisecond); err != nil {
			t.Fatal(err)
		}
		waitRealtimeKeys(t, c, 0)
	})
	t.Run("earlier deadline resets timer", func(t *testing.T) {
		c := New[int, int]()
		t.Cleanup(c.Close)
		if err := c.Set(1, 1, time.Hour); err != nil {
			t.Fatal(err)
		}
		// Observe scheduling of the long timer before replacing its deadline.
		waitRealtimeWorker(t, c)
		if err := c.Set(2, 2, 20*time.Millisecond); err != nil {
			t.Fatal(err)
		}
		waitRealtimeKeys(t, c, 1)
		if got, ok := c.Get(1); !ok || got != 1 {
			t.Fatal("rescheduling removed the long-lived entry")
		}
	})
	t.Run("delete and reinsert", func(t *testing.T) {
		c := New[int, int]()
		t.Cleanup(c.Close)
		if err := c.Set(1, 1, time.Hour); err != nil {
			t.Fatal(err)
		}
		waitRealtimeWorker(t, c)
		c.Delete(1)
		waitRealtimeWorker(t, c)
		if err := c.Set(1, 2, 20*time.Millisecond); err != nil {
			t.Fatal(err)
		}
		waitRealtimeKeys(t, c, 0)
	})
	t.Run("long TTL close", func(t *testing.T) {
		c := New[int, int]()
		t.Cleanup(c.Close)
		if err := c.Set(1, 1, time.Hour); err != nil {
			t.Fatal(err)
		}
		waitRealtimeWorker(t, c)
		closed := make(chan struct{})
		go func() { c.Close(); close(closed) }()
		select {
		case <-closed:
		case <-time.After(5 * time.Second):
			t.Fatal("Close waited for the long TTL")
		}
		select {
		case <-c.done:
		default:
			t.Fatal("Close returned before worker exit")
		}
	})
}

func waitRealtimeKeys(t *testing.T, c *Cache[int, int], want int) {
	t.Helper()
	waitRealtime(t, func() bool {
		c.mu.Lock()
		defer c.mu.Unlock()
		return itemCount(c) == want && len(c.expirations) == want
	})
}

// A drained wake channel shows that the real worker has received the schedule
// change. It may still be about to recheck the heap; either ordering is valid.
func waitRealtimeWorker(t *testing.T, c *Cache[int, int]) {
	t.Helper()
	waitRealtime(t, func() bool { return len(c.wake) == 0 })
}

func waitRealtime(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("real cleanup worker did not reach the expected state")
		}
		time.Sleep(time.Millisecond)
	}
}
