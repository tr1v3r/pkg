package notion

import (
	"context"
	"fmt"

	"golang.org/x/time/rate"
)

const (
	notionAPIHostScheme = "https"
	notionAPIHost       = "api.notion.com"
	apiBasePath         = "v1"
)

func notionAPI() string {
	return fmt.Sprintf("%s://%s/%s", notionAPIHostScheme, notionAPIHost, apiBasePath)
}

func defaultLimiter() *rate.Limiter {
	return rate.NewLimiter(rateLimit, 4*rateLimit)
}

// Manager is the top-level facade for the Notion API.
type Manager struct {
	Database DatabaseAPI
	Page     PageAPI
	Block    BlockAPI
	Search   SearchAPI
	User     UserAPI
	Comment  CommentAPI

	client *notionClient
}

// NewManager creates a Manager with the given Notion API version and token.
func NewManager(version, token string) *Manager {
	limiter := defaultLimiter()
	client := newNotionClient(version, token, limiter)

	return &Manager{
		Database: &DatabaseManager{client: client},
		Page:     &PageManager{client: client},
		Block:    &BlockManager{client: client},
		Search:   &SearchManager{client: client},
		User:     &UserManager{client: client},
		Comment:  &CommentManager{client: client},
		client:   client,
	}
}

// Set updates the Notion API version and token.
func (mgr *Manager) Set(version, token string) {
	mgr.client.version = version
	mgr.client.token = token
}

// WithContext returns a new Manager with the given context.
func (mgr Manager) WithContext(ctx context.Context) *Manager {
	mgr.client = &notionClient{
		version: mgr.client.version,
		token:   mgr.client.token,
		limiter: mgr.client.limiter,
	}
	return &mgr
}

// WithLimiter returns a new Manager with the given rate limiter.
func (mgr Manager) WithLimiter(limiter *rate.Limiter) *Manager {
	mgr.client = &notionClient{
		version: mgr.client.version,
		token:   mgr.client.token,
		limiter: limiter,
	}
	return &mgr
}
