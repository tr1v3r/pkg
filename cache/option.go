package cache

import "time"

// DefaultCleanupBatchSize is the default maximum number of expired entries removed
// by one cleanup pass.
const DefaultCleanupBatchSize = 128

// DefaultCleanupBatchBudget is the default maximum wall-clock time one cleanup pass
// spends removing expired entries.
const DefaultCleanupBatchBudget = time.Millisecond

// cleanupBatchConfig bounds a single cleanup pass. Zero or negative values disable
// the respective limit.
type cleanupBatchConfig struct {
	size   int           // entries removed per pass; <= 0 means unlimited
	budget time.Duration // wall time per pass; <= 0 means unlimited
}

// defaultCleanupBatch returns the default cleanup limits.
func defaultCleanupBatch() cleanupBatchConfig {
	return cleanupBatchConfig{size: DefaultCleanupBatchSize, budget: DefaultCleanupBatchBudget}
}

// exhausted reports whether a pass that already removed removed entries and
// spent elapsed must stop: its entry or time allowance is spent. Callers keep
// the pass going when nothing was removed yet so cleanup always progresses.
func (b cleanupBatchConfig) exhausted(removed int, elapsed time.Duration) bool {
	return (b.size > 0 && removed >= b.size) ||
		(b.budget > 0 && elapsed >= b.budget)
}

// Option configures a Cache during New, before its cleaner starts.
type Option[K comparable, V any] func(*Cache[K, V])

// WithCleanupBatchSize sets the maximum number of expired entries one cleanup pass
// removes before stopping. A nonpositive n removes the limit.
func WithCleanupBatchSize[K comparable, V any](n int) Option[K, V] {
	return func(c *Cache[K, V]) { c.cleanupBatch.size = n }
}

// WithCleanupBatchBudget sets the maximum wall-clock time one cleanup pass spends
// removing expired entries before stopping. A nonpositive d removes the limit.
func WithCleanupBatchBudget[K comparable, V any](d time.Duration) Option[K, V] {
	return func(c *Cache[K, V]) { c.cleanupBatch.budget = d }
}
