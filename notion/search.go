package notion

// NewSearchManager return a new search manager
func NewSearchManager(version, token string) *SearchManager {
	return &SearchManager{baseInfo: &baseInfo{
		NotionVersion: version,
		BearerToken:   token,
	}}
}

// SearchManager ...
type SearchManager struct {
	*baseInfo
}
