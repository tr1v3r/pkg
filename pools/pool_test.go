package pools

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewPool(t *testing.T) {
	size := 5
	pool := NewPool(size)

	assert.NotNil(t, pool)
	assert.Equal(t, size, pool.Size())
	assert.Equal(t, 0, pool.Num())
}

func TestPoolZeroSize(t *testing.T) {
	size := 0
	pool := NewPool(size)

	assert.NotNil(t, pool)
	assert.Equal(t, 0, pool.Size())
	assert.Equal(t, 0, pool.Num())
}

func TestPoolNegativeSize(t *testing.T) {
	size := -1
	pool := NewPool(size)

	assert.NotNil(t, pool)
	assert.Equal(t, 0, pool.Size())
	assert.Equal(t, 0, pool.Num())
}

func TestPoolWaitDone(t *testing.T) {
	size := 2
	pool := NewPool(size)

	// Test basic Wait and Done
	assert.Equal(t, 0, pool.Num())

	pool.Wait()
	assert.Equal(t, 1, pool.Num())

	pool.Done()
	assert.Equal(t, 0, pool.Num())
}

func TestPoolConcurrent(t *testing.T) {
	size := 3
	pool := NewPool(size)

	done := make(chan bool, size)

	// Start multiple goroutines
	for i := 0; i < size; i++ {
		go func(index int) {
			pool.Wait()
			// Simulate some work
			time.Sleep(10 * time.Millisecond)
			done <- true
			pool.Done()
		}(i)
	}

	// Wait for all goroutines to complete
	count := 0
	for i := 0; i < size; i++ {
		select {
		case <-done:
			count++
		case <-time.After(1 * time.Second):
			t.Errorf("Timeout waiting for goroutine %d", i)
		}
	}

	assert.Equal(t, size, count)
	assert.Equal(t, 0, pool.Num())
}

func TestPoolWaitAll(t *testing.T) {
	size := 3
	pool := NewPool(size)

	// Start goroutines that take tokens
	for i := 0; i < size; i++ {
		go func(index int) {
			pool.Wait()
			time.Sleep(50 * time.Millisecond) // Simulate work
			pool.Done()
		}(i)
	}

	// Wait for all tokens to be released
	pool.WaitAll()
	assert.Equal(t, 0, pool.Num())
}

func TestPoolAsyncWait(t *testing.T) {
	size := 2
	pool := NewPool(size)

	// Test AsyncWait
	sig := pool.AsyncWait()
	assert.NotNil(t, sig)

	// Wait for the signal
	select {
	case <-sig:
		// Received signal, token should be taken
		assert.Equal(t, 1, pool.Num())
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for AsyncWait signal")
	}

	// Return the token
	pool.Done()
	assert.Equal(t, 0, pool.Num())
}

func TestPoolAsyncWaitAll(t *testing.T) {
	size := 2
	pool := NewPool(size)

	// Take some tokens
	pool.Wait()
	pool.Wait()

	// Start AsyncWaitAll
	sig := pool.AsyncWaitAll()
	assert.NotNil(t, sig)

	// Return tokens after a delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		pool.Done()
		pool.Done()
	}()

	// Wait for all tokens to be returned
	select {
	case <-sig:
		// All tokens returned
		assert.Equal(t, 0, pool.Num())
	case <-time.After(200 * time.Millisecond):
		t.Error("Timeout waiting for AsyncWaitAll signal")
	}
}

func TestPoolNilPool(t *testing.T) {
	var p *pool

	// Test methods on nil pool that are safe
	assert.Equal(t, 0, p.Num())
	assert.Equal(t, 0, p.Size())

	// Async methods are safe for nil pool
	sig1 := p.AsyncWait()
	sig2 := p.AsyncWaitAll()
	assert.NotNil(t, sig1)
	assert.NotNil(t, sig2)
}

func TestPoolAsyncWaitWithNilPool(t *testing.T) {
	var p *pool

	// Test AsyncWait methods on nil pool
	sig1 := p.AsyncWait()
	assert.NotNil(t, sig1)

	// Should receive signal immediately for nil pool
	select {
	case <-sig1:
		// Expected
	case <-time.After(10 * time.Millisecond):
		t.Error("Timeout waiting for AsyncWait signal on nil pool")
	}

	sig2 := p.AsyncWaitAll()
	assert.NotNil(t, sig2)

	// Should receive signal immediately for nil pool
	select {
	case <-sig2:
		// Expected
	case <-time.After(10 * time.Millisecond):
		t.Error("Timeout waiting for AsyncWaitAll signal on nil pool")
	}
}