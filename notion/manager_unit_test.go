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
	assert.NotNil(t, mgr.Database)
	assert.NotNil(t, mgr.Page)
	assert.NotNil(t, mgr.Block)
	assert.NotNil(t, mgr.Search)
	assert.NotNil(t, mgr.User)
	assert.NotNil(t, mgr.Comment)
}

func TestManagerClient(t *testing.T) {
	version := "2022-06-28"
	token := "test-token"

	mgr := NewManager(version, token)

	assert.NotNil(t, mgr.client)
	assert.Equal(t, version, mgr.client.version)
	assert.Equal(t, token, mgr.client.token)
	assert.NotNil(t, mgr.client.limiter)
}

func TestManagerSet(t *testing.T) {
	version1 := "2022-06-28"
	token1 := "test-token-1"
	version2 := "2022-10-01"
	token2 := "test-token-2"

	mgr := NewManager(version1, token1)
	assert.Equal(t, version1, mgr.client.version)
	assert.Equal(t, token1, mgr.client.token)

	mgr.Set(version2, token2)
	assert.Equal(t, version2, mgr.client.version)
	assert.Equal(t, token2, mgr.client.token)
}

func TestDatabaseManagerCreation(t *testing.T) {
	version := "2022-06-28"
	token := "test-token"

	dbMgr := NewDatabaseManager(version, token)

	assert.NotNil(t, dbMgr)
	assert.NotNil(t, dbMgr.client)
	assert.Equal(t, version, dbMgr.client.version)
	assert.Equal(t, token, dbMgr.client.token)
}

func TestPageManagerCreation(t *testing.T) {
	version := "2022-06-28"
	token := "test-token"

	pageMgr := NewPageManager(version, token)

	assert.NotNil(t, pageMgr)
	assert.NotNil(t, pageMgr.client)
	assert.Equal(t, version, pageMgr.client.version)
	assert.Equal(t, token, pageMgr.client.token)
}

func TestBlockManagerCreation(t *testing.T) {
	version := "2022-06-28"
	token := "test-token"

	blockMgr := NewBlockManager(version, token)

	assert.NotNil(t, blockMgr)
	assert.NotNil(t, blockMgr.client)
	assert.Equal(t, version, blockMgr.client.version)
	assert.Equal(t, token, blockMgr.client.token)
}

func TestSearchManagerCreation(t *testing.T) {
	version := "2022-06-28"
	token := "test-token"

	searchMgr := NewSearchManager(version, token)

	assert.NotNil(t, searchMgr)
	assert.NotNil(t, searchMgr.client)
	assert.Equal(t, version, searchMgr.client.version)
	assert.Equal(t, token, searchMgr.client.token)
}

func TestUserManagerCreation(t *testing.T) {
	version := "2022-06-28"
	token := "test-token"

	userMgr := NewUserManager(version, token)

	assert.NotNil(t, userMgr)
	assert.NotNil(t, userMgr.client)
}

func TestCommentManagerCreation(t *testing.T) {
	version := "2022-06-28"
	token := "test-token"

	commentMgr := NewCommentManager(version, token)

	assert.NotNil(t, commentMgr)
	assert.NotNil(t, commentMgr.client)
}

func TestNotionAPIConstants(t *testing.T) {
	assert.Equal(t, "error", ErrorObjectType)
	assert.Equal(t, 3, rateLimit)
}

func TestNotionAPIHost(t *testing.T) {
	expected := "https://api.notion.com/v1"
	actual := notionAPI()
	assert.Equal(t, expected, actual)
}
