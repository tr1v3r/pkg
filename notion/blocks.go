package notion

// NewBlockManager return a new database manager
func NewBlockManager(version, token string) *BlockManager {
	return &BlockManager{baseInfo: &baseInfo{
		NotionVersion: version,
		BearerToken:   token,
	}}
}

// BlockManager ...
type BlockManager struct {
	*baseInfo

	ID string
}

func (bm BlockManager) WithID(id string) *BlockManager {
	bm.ID = id
	return &bm
}
