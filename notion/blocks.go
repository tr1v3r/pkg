package notion

import (
	"context"

	"golang.org/x/time/rate"
)

// NewBlockManager return a new database manager
func NewBlockManager(version, token string) *BlockManager {
	return &BlockManager{baseInfo: &baseInfo{
		NotionVersion: version,
		BearerToken:   token,
	}, ctx: context.Background(), limiter: rate.NewLimiter(3, 12)}
}

// BlockManager ...
type BlockManager struct {
	*baseInfo

	ctx     context.Context
	id      string
	limiter *rate.Limiter
}

// WithContext set Context
func (bm BlockManager) WithContext(ctx context.Context) *BlockManager {
	bm.ctx = ctx
	return &bm
}

// WithID set block id
func (bm BlockManager) WithID(id string) *BlockManager {
	bm.id = id
	return &bm
}

// WithLimiter with limiiter
func (bm BlockManager) WithLimiter(limiter *rate.Limiter) *BlockManager {
	bm.limiter = limiter
	return &bm
}
