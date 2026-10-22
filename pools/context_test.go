package pools

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// Signal when WaitContext reaches its select, after registering with the WaitGroup.
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

func TestUnlimitedCompletion(t *testing.T) {
	for _, async := range []bool{false, true} {
		name := "Wait"
		if async {
			name = "AsyncWait"
		}
		t.Run(name, func(t *testing.T) {
			p := NewPool(-1)
			if async {
				awaitSignal(t, p.AsyncWait())
			} else {
				p.Wait()
			}
			done := p.AsyncWaitAll()
			select {
			case <-done:
				t.Fatal("completion returned with outstanding work")
			case <-time.After(10 * time.Millisecond):
			}
			p.Done()
			awaitSignal(t, done)
			// A second batch must also be tracked.
			p.Wait()
			p.Done()
			p.WaitAll()
		})
	}
}

func TestWaitContextPreCancelled(t *testing.T) {
	for _, size := range []int{1, -1} {
		p := NewPool(size)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := p.WaitContext(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v", err)
		}
		if p.Num() != 0 {
			t.Fatal("cancelled acquisition retained a token")
		}
		awaitSignal(t, p.AsyncWaitAll())
	}
}

func TestWaitContextCancelledWhileFull(t *testing.T) {
	p := NewPool(1)
	p.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	observed := &observedContext{Context: ctx, selecting: make(chan struct{})}
	result := make(chan error, 1)
	go func() { result <- p.WaitContext(observed) }()
	awaitSignal(t, observed.selecting)
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if p.Num() != 1 {
		t.Fatal("cancellation released another worker's token")
	}
	p.Done()
	awaitSignal(t, p.AsyncWaitAll())
	if err := p.WaitContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	p.Done()
	p.WaitAll()
}

func TestWaitContextDeadline(t *testing.T) {
	p := NewPool(1)
	p.Wait()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := p.WaitContext(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
	p.Done()
	awaitSignal(t, p.AsyncWaitAll())
}

func TestWaitContextSuccessSurvivesCancellation(t *testing.T) {
	for _, size := range []int{1, -1} {
		p := NewPool(size)
		ctx, cancel := context.WithCancel(context.Background())
		if err := p.WaitContext(ctx); err != nil {
			cancel()
			t.Fatal(err)
		}
		cancel()
		if size > 0 && p.Num() != 1 {
			t.Fatal("cancellation revoked successful acquisition")
		}
		p.Done()
		awaitSignal(t, p.AsyncWaitAll())
	}
}

func TestWaitContextCancelReleaseRace(t *testing.T) {
	for i := 0; i < 200; i++ {
		p := NewPool(1)
		p.Wait()
		ctx, cancel := context.WithCancel(context.Background())
		observed := &observedContext{Context: ctx, selecting: make(chan struct{})}
		result := make(chan error, 1)
		go func() { result <- p.WaitContext(observed) }()
		awaitSignal(t, observed.selecting)
		var racers sync.WaitGroup
		racers.Add(2)
		go func() { defer racers.Done(); cancel() }()
		go func() { defer racers.Done(); p.Done() }()
		err := <-result
		racers.Wait()
		if err == nil {
			p.Done()
		} else if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		awaitSignal(t, p.AsyncWaitAll())
		if p.Num() != 0 {
			t.Fatal("token leaked after cancellation/release race")
		}
	}
}
