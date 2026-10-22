package pools

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// Signal when Acquire evaluates the cancellation case in its select.
type observedContext struct {
	context.Context
	once      sync.Once
	selecting chan struct{}
}

func (c *observedContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.selecting) })
	return c.Context.Done()
}

func awaitSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for completion")
	}
}

func TestAcquirePreCancelled(t *testing.T) {
	for _, size := range []int{1, -1} {
		p := NewPool(size)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := p.Acquire(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v", err)
		}
		if p.Num() != 0 {
			t.Fatal("cancelled acquisition retained a token")
		}
	}
}

func TestAcquireCancelledWhileFull(t *testing.T) {
	p := NewPool(1)
	acquire(t, p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	observed := &observedContext{Context: ctx, selecting: make(chan struct{})}
	result := make(chan error, 1)
	go func() { result <- p.Acquire(observed) }()
	awaitSignal(t, observed.selecting)
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if p.Num() != 1 {
		t.Fatal("cancellation released another worker's token")
	}
	p.Release()
	if err := p.Acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
	p.Release()
}

func TestAcquireDeadline(t *testing.T) {
	p := NewPool(1)
	acquire(t, p)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := p.Acquire(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
	p.Release()
}

func TestAcquireSuccessSurvivesCancellation(t *testing.T) {
	for _, size := range []int{1, -1} {
		p := NewPool(size)
		ctx, cancel := context.WithCancel(context.Background())
		if err := p.Acquire(ctx); err != nil {
			cancel()
			t.Fatal(err)
		}
		cancel()
		if size > 0 && p.Num() != 1 {
			t.Fatal("cancellation revoked successful acquisition")
		}
		p.Release()
	}
}

func TestAcquireCancelReleaseRace(t *testing.T) {
	for i := 0; i < 200; i++ {
		p := NewPool(1)
		acquire(t, p)
		ctx, cancel := context.WithCancel(context.Background())
		observed := &observedContext{Context: ctx, selecting: make(chan struct{})}
		result := make(chan error, 1)
		go func() { result <- p.Acquire(observed) }()
		awaitSignal(t, observed.selecting)
		var racers sync.WaitGroup
		racers.Add(2)
		go func() { defer racers.Done(); cancel() }()
		go func() { defer racers.Done(); p.Release() }()
		err := <-result
		racers.Wait()
		if err == nil {
			p.Release()
		} else if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		if p.Num() != 0 {
			t.Fatal("token leaked after cancellation/release race")
		}
	}
}
