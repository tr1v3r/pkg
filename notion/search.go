package notion

import (
	"context"

	"golang.org/x/time/rate"
)

// NewSearchManager return a new search manager
func NewSearchManager(version, token string) *SearchManager {
	return &SearchManager{baseInfo: &baseInfo{
		NotionVersion: version,
		BearerToken:   token,
	}, ctx: context.Background()}
}

// SearchManager ...
type SearchManager struct {
	*baseInfo

	ctx     context.Context
	limiter *rate.Limiter
}

// WithContext set Context
func (sm SearchManager) WithContext(ctx context.Context) *SearchManager {
	sm.ctx = ctx
	return &sm
}

// WithLimiter with limiiter
func (sm SearchManager) WithLimiter(limiter *rate.Limiter) *SearchManager {
	sm.limiter = limiter
	return &sm
}
