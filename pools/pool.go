package pools

import (
	"context"
	"sync"
)

// Pool goroutine pool
type Pool interface {
	// Wait 等待令牌
	Wait()
	// WaitContext waits for a token or cancellation. On success, call Done once.
	// On error, no token or completion count is retained.
	WaitContext(ctx context.Context) error
	// AsyncWait 异步等待令牌
	//
	// Deprecated: use WaitContext for cancellable acquisition.
	AsyncWait() <-chan struct{}
	// Done 归还令牌
	Done()
	// Num 当前发放的令牌书
	Num() int
	// Size 总令牌数
	Size() int

	// WaitAll 同步等待令牌全部归还
	WaitAll()
	// AsyncWaitAll 异步等待令牌全部归还
	AsyncWaitAll() <-chan struct{}
}

type pool struct {
	pool chan struct{}

	wg sync.WaitGroup
}

// Init initializes the pool with the given capacity.
func (p *pool) Init(size int) {
	if size == 0 {
		panic("pools: size must not be zero")
	}
	if size > 0 {
		p.pool = make(chan struct{}, size)
	}
}

// Wait blocks until a token is acquired and registers work for WaitAll.
func (p *pool) Wait() {
	_ = p.WaitContext(context.Background())
}

// WaitContext acquires a token or returns ctx.Err(). It does not start a goroutine.
// An already-cancelled context is rejected before registering work. If cancellation
// races with available capacity, either outcome is possible: only a nil return
// requires Done. Cancellation after success does not release the token.
// As with Wait, submit work before calling WaitAll, including in unlimited mode.
func (p *pool) WaitContext(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.wg.Add(1)
	if p.pool == nil {
		return nil
	}
	select {
	case p.pool <- struct{}{}:
		return nil
	case <-ctx.Done():
		p.wg.Done()
		return ctx.Err()
	}
}

// AsyncWait closes its signal after acquiring a token. It cannot be cancelled.
//
// Deprecated: use WaitContext for cancellable acquisition.
func (p *pool) AsyncWait() <-chan struct{} {
	sig := make(chan struct{})
	if p == nil {
		close(sig)
		return sig
	}

	go func() {
		p.Wait()
		close(sig)
	}()
	return sig
}

// Done releases a pool token.
func (p *pool) Done() {
	if p != nil {
		if p.pool != nil {
			<-p.pool
		}
		p.wg.Done()
	}
}

// Num returns the number of held bounded-pool tokens, or zero in unlimited mode.
func (p *pool) Num() int {
	if p != nil && p.pool != nil {
		return len(p.pool)
	}
	return 0
}

// Size returns the total token capacity.
func (p *pool) Size() int {
	if p != nil && p.pool != nil {
		return cap(p.pool)
	}
	return 0
}

// WaitAll blocks until every issued token has been returned.
// Like sync.WaitGroup, all Wait() calls that begin while the pool is idle
// must happen before WaitAll is called, or WaitAll may return early.
func (p *pool) WaitAll() {
	if p != nil {
		p.wg.Wait()
	}
}

// AsyncWaitAll closes its signal when WaitAll returns.
func (p *pool) AsyncWaitAll() <-chan struct{} {
	sig := make(chan struct{})
	go func() {
		p.WaitAll()
		close(sig)
	}()
	return sig
}
