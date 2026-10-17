package cache

import "time"

// DefaultCleanupBatchSize is the default maximum number of expired entries removed
// by one cleanup pass.
const DefaultCleanupBatchSize = 128

// DefaultCleanupBatchBudget is the default maximum wall-clock time one cleanup pass
// spends removing expired entries.
const DefaultCleanupBatchBudget = time.Millisecond

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
