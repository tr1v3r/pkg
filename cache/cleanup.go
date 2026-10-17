package cache

import "time"

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
