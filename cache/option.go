package cache

import "time"

// DefaultBatchSize is the default maximum number of expired entries removed
// by one cleanup pass.
const DefaultBatchSize = 128

// DefaultBatchBudget is the default maximum wall-clock time one cleanup pass
// spends removing expired entries.
const DefaultBatchBudget = time.Millisecond

// batchConfig bounds a single cleanup pass. Zero or negative values disable
// the respective limit.
type batchConfig struct {
	size   int           // entries removed per pass; <= 0 means unlimited
	budget time.Duration // wall time per pass; <= 0 means unlimited
}

// defaultBatch returns the default cleanup limits.
func defaultBatch() batchConfig {
	return batchConfig{size: DefaultBatchSize, budget: DefaultBatchBudget}
}

// exhausted reports whether a pass that already removed removed entries and
// spent elapsed must stop: its entry or time allowance is spent. Callers keep
// the pass going when nothing was removed yet so cleanup always progresses.
func (b batchConfig) exhausted(removed int, elapsed time.Duration) bool {
	if b.size > 0 && removed >= b.size {
		return true
	}
	return b.budget > 0 && elapsed >= b.budget
}

// Option customizes the limits of a single expiration cleanup pass.
type Option func(*batchConfig)

// WithBatchSize sets the maximum number of expired entries one cleanup pass
// removes before stopping. A nonpositive n removes the limit.
func WithBatchSize(n int) Option {
	return func(b *batchConfig) { b.size = n }
}

// WithBatchBudget sets the maximum wall-clock time one cleanup pass spends
// removing expired entries before stopping. A nonpositive d removes the limit.
func WithBatchBudget(d time.Duration) Option {
	return func(b *batchConfig) { b.budget = d }
}
