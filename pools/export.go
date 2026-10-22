// Package pools : goroutine pools
package pools

// NewPool returns a pool. Positive size limits concurrency; negative size permits
// unlimited concurrency while still tracking completion. Zero size panics.
func NewPool(size int) Pool {
	var p pool
	p.Init(size)
	return &p
}
