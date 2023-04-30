package notion

import (
	"fmt"

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
	return &Manager{
		DatabaseManager: NewDatabaseManager(version, token),
		PageManager:     NewPageManager(version, token),
		BlockManager:    NewBlockManager(version, token),
		SearchManager:   NewSearchManager(version, token),
		baseInfo: &baseInfo{
			NotionVersion: version,
			BearerToken:   token,
		},
	}
}

func (i *Manager) Set(version, token string) {
	i.DatabaseManager.Set(version, token)
	i.PageManager.Set(version, token)
	i.BlockManager.Set(version, token)
	i.SearchManager.Set(version, token)
	i.Set(version, token)
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
