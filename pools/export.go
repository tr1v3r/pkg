// Package pools provides context-aware concurrency limits without worker management
// or task completion tracking.
package pools

// NewPool returns a concurrency limiter. Positive size bounds concurrency;
// negative size disables the limit. Zero size panics.
func NewPool(size int) Pool {
	if size == 0 {
		panic("pools: size must not be zero")
	}
	p := &pool{}
	if size > 0 {
		p.tokens = make(chan struct{}, size)
	}
	return p
}
