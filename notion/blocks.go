package notion

import "context"

// NewBlockManager return a new database manager
func NewBlockManager(version, token string) *BlockManager {
	return &BlockManager{baseInfo: &baseInfo{
		NotionVersion: version,
		BearerToken:   token,
	}, ctx: context.Background()}
}

// BlockManager ...
type BlockManager struct {
	*baseInfo

	ID string

	ctx context.Context
}

func (bm BlockManager) WithID(id string) *BlockManager {
	bm.ID = id
	return &bm
}

// WithContext set Context
func (bm BlockManager) WithContext(ctx context.Context) *BlockManager {
	bm.ctx = ctx
	return &bm
}
