package pools

import "context"

// Pool limits concurrent work. It does not start workers or track task completion.
// Use NewPool to construct a Pool; callers must not pass a nil context.
type Pool interface {
	// Acquire waits for capacity or cancellation. Call Release exactly once after
	// success; on error no token is retained. Cancellation after success does not
	// release the token. If cancellation races with capacity, either may win.
	Acquire(ctx context.Context) error
	// Release returns one token. In bounded mode it panics if no token is held.
	// In unlimited mode it is a no-op. Tokens are not tied to a goroutine or owner.
	Release()
	// Num is a snapshot of held bounded-pool tokens, or zero in unlimited mode.
	// It is not a completion barrier and excludes pending acquisitions.
	Num() int
	// Size is the concurrency limit, or zero in unlimited mode.
	Size() int
}

type pool struct {
	tokens chan struct{}
}

func (p *pool) Acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.tokens == nil {
		return nil
	}
	select {
	case p.tokens <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *pool) Release() {
	if p.tokens == nil {
		return
	}
	select {
	case <-p.tokens:
	default:
		panic("pools: release without acquisition")
	}
}

func (p *pool) Num() int { return len(p.tokens) }

func (p *pool) Size() int { return cap(p.tokens) }
