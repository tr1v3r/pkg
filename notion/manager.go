package notion

import (
	"context"
	"fmt"

	"golang.org/x/time/rate"

	"github.com/riverchu/pkg/fetch"
)

const (
	notionAPIHostScheme = "https"
	notionAPIHost       = "api.notion.com"
	apiBasePath         = "v1"
)

// operateType define operate type
type operateType string

const (
	createOp       operateType = "create"
	queryOp        operateType = "query"
	retrieveOp     operateType = "retreive"
	retrievePropOp operateType = "retrieveProp"
	updateOp       operateType = "update"
)

// notionAPI return notion api url
func notionAPI() string {
	return fmt.Sprintf("%s://%s/%s", notionAPIHostScheme, notionAPIHost, apiBasePath)
}

// Manager is a manager for notion
type Manager struct {
	*DatabaseManager
	*PageManager
	*BlockManager
	*SearchManager

	*baseInfo
}

// NewManager return a new notion manager
func NewManager(version, token string) *Manager {
	limiter := rate.NewLimiter(3, 12)
	return &Manager{
		DatabaseManager: NewDatabaseManager(version, token).WithLimiter(limiter),
		PageManager:     NewPageManager(version, token).WithLimiter(limiter),
		BlockManager:    NewBlockManager(version, token).WithLimiter(limiter),
		SearchManager:   NewSearchManager(version, token).WithLimiter(limiter),
		baseInfo: &baseInfo{
			NotionVersion: version,
			BearerToken:   token,
		},
	}
}

// Set set notion version and token
func (mgr *Manager) Set(version, token string) {
	mgr.DatabaseManager.Set(version, token)
	mgr.PageManager.Set(version, token)
	mgr.BlockManager.Set(version, token)
	mgr.SearchManager.Set(version, token)
	mgr.baseInfo.Set(version, token)
}

// WithContext set context for notion manager
func (mgr Manager) WithContext(ctx context.Context) *Manager {
	mgr.DatabaseManager = mgr.DatabaseManager.WithContext(ctx)
	mgr.PageManager = mgr.PageManager.WithContext(ctx)
	mgr.BlockManager = mgr.BlockManager.WithContext(ctx)
	mgr.SearchManager = mgr.SearchManager.WithContext(ctx)
	return &mgr
}

// WithLimiter set limiter for notion manager
func (mgr Manager) WithLimiter(limiter *rate.Limiter) *Manager {
	mgr.DatabaseManager = mgr.DatabaseManager.WithLimiter(limiter)
	mgr.PageManager = mgr.PageManager.WithLimiter(limiter)
	mgr.BlockManager = mgr.BlockManager.WithLimiter(limiter)
	mgr.SearchManager = mgr.SearchManager.WithLimiter(limiter)
	return &mgr
}

type baseInfo struct {
	NotionVersion string
	BearerToken   string
}

func (i *baseInfo) Headers() []fetch.RequestOption {
	return []fetch.RequestOption{
		fetch.WithHeader("Notion-Version", i.NotionVersion),
		fetch.WithAuthToken("Bearer " + i.BearerToken),
		fetch.WithContentTypeJSON(),
	}
}

func (i *baseInfo) Set(version, token string) {
	i.NotionVersion = version
	i.BearerToken = token
}
