package notion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	version := "2022-06-28"
	token := "test-token"

	mgr := NewManager(version, token)

	assert.NotNil(t, mgr)
	assert.NotNil(t, mgr.DatabaseManager)
	assert.NotNil(t, mgr.PageManager)
	assert.NotNil(t, mgr.BlockManager)
	assert.NotNil(t, mgr.SearchManager)
}

func TestManagerHeaders(t *testing.T) {
	version := "2022-06-28"
	token := "test-token"

	mgr := NewManager(version, token)
	headers := mgr.Headers()

	// Headers returns []fetch.RequestOption, so we just verify it's not empty
	assert.NotEmpty(t, headers)
	assert.Len(t, headers, 3) // Should have 3 headers: Notion-Version, Authorization, Content-Type
}

func TestDatabaseManagerCreation(t *testing.T) {
	version := "2022-06-28"
	token := "test-token"

	dbMgr := NewDatabaseManager(version, token)

	assert.NotNil(t, dbMgr)
	assert.NotNil(t, dbMgr.baseInfo)
	assert.Equal(t, version, dbMgr.baseInfo.NotionVersion)
	assert.Equal(t, token, dbMgr.baseInfo.BearerToken)
}

func TestPageManagerCreation(t *testing.T) {
	version := "2022-06-28"
	token := "test-token"

	pageMgr := NewPageManager(version, token)

	assert.NotNil(t, pageMgr)
	assert.NotNil(t, pageMgr.baseInfo)
	assert.Equal(t, version, pageMgr.baseInfo.NotionVersion)
	assert.Equal(t, token, pageMgr.baseInfo.BearerToken)
}

func TestBlockManagerCreation(t *testing.T) {
	version := "2022-06-28"
	token := "test-token"

	blockMgr := NewBlockManager(version, token)

	assert.NotNil(t, blockMgr)
	assert.NotNil(t, blockMgr.baseInfo)
	assert.Equal(t, version, blockMgr.baseInfo.NotionVersion)
	assert.Equal(t, token, blockMgr.baseInfo.BearerToken)
}

func TestSearchManagerCreation(t *testing.T) {
	version := "2022-06-28"
	token := "test-token"

	searchMgr := NewSearchManager(version, token)

	assert.NotNil(t, searchMgr)
	assert.NotNil(t, searchMgr.baseInfo)
	assert.Equal(t, version, searchMgr.baseInfo.NotionVersion)
	assert.Equal(t, token, searchMgr.baseInfo.BearerToken)
}

func TestManagerSet(t *testing.T) {
	version1 := "2022-06-28"
	token1 := "test-token-1"
	version2 := "2022-10-01"
	token2 := "test-token-2"

	mgr := NewManager(version1, token1)
	assert.Equal(t, version1, mgr.baseInfo.NotionVersion)
	assert.Equal(t, token1, mgr.baseInfo.BearerToken)

	mgr.Set(version2, token2)
	assert.Equal(t, version2, mgr.baseInfo.NotionVersion)
	assert.Equal(t, token2, mgr.baseInfo.BearerToken)
}

func TestDatabaseManagerWithID(t *testing.T) {
	version := "2022-06-28"
	token := "test-token"
	databaseID := "test-database-id"

	mgr := NewManager(version, token)
	dbMgr := mgr.DatabaseManager.WithID(databaseID)

	assert.NotNil(t, dbMgr)
	// The WithID method should return a new manager with updated ID
	assert.NotEqual(t, mgr.DatabaseManager, dbMgr) // Different instances
	// Verify the new manager has the updated baseInfo (they share baseInfo)
	assert.Equal(t, mgr.DatabaseManager.baseInfo, dbMgr.baseInfo)
}

func TestPageManagerWithID(t *testing.T) {
	version := "2022-06-28"
	token := "test-token"
	pageID := "test-page-id"

	mgr := NewManager(version, token)
	pageMgr := mgr.PageManager.WithID(pageID)

	assert.NotNil(t, pageMgr)
	// The WithID method should return a new manager
	assert.NotEqual(t, mgr.PageManager, pageMgr)
}

func TestBlockManagerWithID(t *testing.T) {
	version := "2022-06-28"
	token := "test-token"
	blockID := "test-block-id"

	mgr := NewManager(version, token)
	blockMgr := mgr.BlockManager.WithID(blockID)

	assert.NotNil(t, blockMgr)
	// The WithID method should return a new manager
	assert.NotEqual(t, mgr.BlockManager, blockMgr)
}

func TestSearchManagerWithLimiter(t *testing.T) {
	version := "2022-06-28"
	token := "test-token"

	mgr := NewManager(version, token)
	searchMgr := mgr.SearchManager.WithLimiter(nil) // Can pass nil for test

	assert.NotNil(t, searchMgr)
	// The WithLimiter method should return a new manager
	assert.NotEqual(t, mgr.SearchManager, searchMgr)
}

func TestNotionAPIConstants(t *testing.T) {
	// Test the constant we defined for lint fixes
	assert.Equal(t, "error", ErrorObjectType)

	// Test rate limit constant from object.go
	assert.Equal(t, 3, rateLimit)
}

func TestNotionAPIHost(t *testing.T) {
	// Test the API host generation
	expected := "https://api.notion.com/v1"
	actual := notionAPI()
	assert.Equal(t, expected, actual)
}
