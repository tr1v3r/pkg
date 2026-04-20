package notion

import (
	"context"
	"fmt"

	"github.com/tr1v3r/pkg/log"
)

// SearchManager implements SearchAPI.
type SearchManager struct {
	client *notionClient
}

// NewSearchManager creates a SearchManager with default settings.
func NewSearchManager(version, token string) *SearchManager {
	return &SearchManager{
		client: newNotionClient(version, token, defaultLimiter()),
	}
}

// Search searches pages and databases in the workspace.
// POST /v1/search
func (sm *SearchManager) Search(ctx context.Context, query string, filter *SearchFilter) (*ListResponse[SearchResult], error) {
	log.CtxDebugf(ctx, "search: %s", query)

	body := map[string]any{}
	if query != "" {
		body["query"] = query
	}
	if filter != nil {
		body["filter"] = filter
	}

	var resp ListResponse[SearchResult]
	if err := sm.client.post(ctx, "/search", body, &resp); err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	return &resp, nil
}
