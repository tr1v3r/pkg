package pools

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func acquire(t *testing.T, p Pool) {
	t.Helper()
	if err := p.Acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestNewPool(t *testing.T) {
	p := NewPool(2)
	assert.Equal(t, 2, p.Size())
	assert.Equal(t, 0, p.Num())
	acquire(t, p)
	assert.Equal(t, 1, p.Num())
	p.Release()
	assert.Equal(t, 0, p.Num())
}

func TestPoolZeroSize(t *testing.T) {
	assert.PanicsWithValue(t, "pools: size must not be zero", func() { NewPool(0) })
}

func TestUnlimitedPool(t *testing.T) {
	p := NewPool(-1)
	for i := 0; i < 100; i++ {
		acquire(t, p)
	}
	assert.Zero(t, p.Size())
	assert.Zero(t, p.Num())
	for i := 0; i < 100; i++ {
		p.Release()
	}
}

func TestReleaseWithoutAcquisition(t *testing.T) {
	p := NewPool(1)
	assert.PanicsWithValue(t, "pools: release without acquisition", p.Release)
	acquire(t, p)
	p.Release()
	assert.PanicsWithValue(t, "pools: release without acquisition", p.Release)
}

func TestAcquireBlocksUntilRelease(t *testing.T) {
	p := NewPool(1)
	acquire(t, p)
	observed := &observedContext{Context: context.Background(), selecting: make(chan struct{})}
	acquired := make(chan struct{})
	go func() {
		if err := p.Acquire(observed); err != nil {
			t.Error(err)
		}
		close(acquired)
	}()
	awaitSignal(t, observed.selecting)
	select {
	case <-acquired:
		t.Fatal("acquired beyond capacity")
	default:
	}
	p.Release()
	awaitSignal(t, acquired)
	assert.Equal(t, 1, p.Num())
	p.Release()
}

func TestConcurrentLimitAndReuse(t *testing.T) {
	p := NewPool(3)
	for batch := 0; batch < 2; batch++ {
		var wg sync.WaitGroup
		var active atomic.Int32
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := p.Acquire(context.Background()); err != nil {
					t.Error(err)
					return
				}
				defer p.Release()
				if n := active.Add(1); n > 3 {
					t.Errorf("active workers exceed capacity: %d", n)
				}
				active.Add(-1)
			}()
		}
		wg.Wait()
		assert.Zero(t, p.Num())
	}
}
