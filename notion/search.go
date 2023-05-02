package notion

import "context"

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

	ctx context.Context
}

// WithContext set Context
func (sm SearchManager) WithContext(ctx context.Context) *SearchManager {
	sm.ctx = ctx
	return &sm
}
