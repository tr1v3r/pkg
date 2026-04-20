package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// DatabaseManager tests
// =============================================================================

func TestDatabaseManager_Create(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		verifyAuth(t, r)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/databases", r.URL.Path)

		body := readBody(t, r)
		require.NotNil(t, body)
		assert.Equal(t, "database_id", body["parent"].(map[string]any)["type"])

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(sampleDatabaseJSON())) //nolint:errcheck
	})

	dm := &DatabaseManager{client: client}
	db, err := dm.Create(testContext(), ParentRef{Type: "database_id", DatabaseID: "db-parent"}, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "database", db.Object)
	assert.Equal(t, "db-123", db.ID)
}

func TestDatabaseManager_Retrieve(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		verifyAuth(t, r)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/databases/db-123", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(sampleDatabaseJSON())) //nolint:errcheck
	})

	dm := &DatabaseManager{client: client}
	db, err := dm.Retrieve(testContext(), "db-123")
	require.NoError(t, err)
	assert.Equal(t, "database", db.Object)
	assert.Equal(t, "db-123", db.ID)
	assert.Equal(t, "https://notion.so/db-123", db.URL)
}

func TestDatabaseManager_Query_SinglePage(t *testing.T) {
	callCount := 0
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		verifyAuth(t, r)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.True(t, strings.HasPrefix(r.URL.Path, "/v1/databases/db-123/query"))

		callCount++
		pages := []map[string]any{
			{
				"object": "page", "id": "page-1",
				"created_time": "2024-01-01T00:00:00Z",
				"created_by":   map[string]any{"object": "user", "id": "user-1"},
			},
		}
		jsonRespond(w, http.StatusOK, paginatedResponse(pages, false, ""))
	})

	dm := &DatabaseManager{client: client}
	pages, err := dm.Query(testContext(), "db-123", nil)
	require.NoError(t, err)
	assert.Len(t, pages, 1)
	assert.Equal(t, "page-1", pages[0].ID)
	assert.Equal(t, 1, callCount)
}

func TestDatabaseManager_Query_MultiplePages(t *testing.T) {
	callCount := 0
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		assert.Equal(t, http.MethodPost, r.Method)

		body := readBody(t, r)
		var pages []map[string]any

		if callCount == 1 {
			// First call: no start_cursor expected.
			_, hasCursor := body["start_cursor"]
			assert.False(t, hasCursor)
			pages = []map[string]any{
				{"object": "page", "id": "page-1", "created_time": "2024-01-01T00:00:00Z",
					"created_by": map[string]any{"object": "user", "id": "u1"}},
				{"object": "page", "id": "page-2", "created_time": "2024-01-01T00:00:00Z",
					"created_by": map[string]any{"object": "user", "id": "u1"}},
			}
			jsonRespond(w, http.StatusOK, paginatedResponse(pages, true, "cursor-abc"))
		} else {
			// Second call: start_cursor should be set.
			assert.Equal(t, "cursor-abc", body["start_cursor"])
			pages = []map[string]any{
				{"object": "page", "id": "page-3", "created_time": "2024-01-01T00:00:00Z",
					"created_by": map[string]any{"object": "user", "id": "u1"}},
			}
			jsonRespond(w, http.StatusOK, paginatedResponse(pages, false, ""))
		}
	})

	dm := &DatabaseManager{client: client}
	pages, err := dm.Query(testContext(), "db-123", &Condition{PageSize: 2})
	require.NoError(t, err)
	assert.Len(t, pages, 3)
	assert.Equal(t, "page-1", pages[0].ID)
	assert.Equal(t, "page-2", pages[1].ID)
	assert.Equal(t, "page-3", pages[2].ID)
	assert.Equal(t, 2, callCount)
}

func TestDatabaseManager_Query_WithFilter(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		body := readBody(t, r)
		assert.NotNil(t, body["filter"])

		pages := []map[string]any{
			{"object": "page", "id": "page-1", "created_time": "2024-01-01T00:00:00Z",
				"created_by": map[string]any{"object": "user", "id": "u1"}},
		}
		jsonRespond(w, http.StatusOK, paginatedResponse(pages, false, ""))
	})

	dm := &DatabaseManager{client: client}
	cond := &Condition{
		PageSize: 100,
		Filter:   &FilterCondition{FilterSingleCondition: FilterSingleCondition{Property: "Status", Select: &SelectFilter{Equals: "Done"}}},
	}
	pages, err := dm.Query(testContext(), "db-123", cond)
	require.NoError(t, err)
	assert.Len(t, pages, 1)
}

func TestDatabaseManager_Update(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		verifyAuth(t, r)
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/v1/databases/db-123", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(sampleDatabaseJSON())) //nolint:errcheck
	})

	dm := &DatabaseManager{client: client}
	payload := strings.NewReader(`{"title": [{"type": "text", "text": {"content": "Updated"}}]}`)
	db, err := dm.Update(testContext(), "db-123", payload)
	require.NoError(t, err)
	assert.Equal(t, "db-123", db.ID)
}

// =============================================================================
// PageManager tests
// =============================================================================

func TestPageManager_Create(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		verifyAuth(t, r)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/pages", r.URL.Path)

		body := readBody(t, r)
		require.NotNil(t, body)
		parent, ok := body["parent"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "database_id", parent["type"])

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(samplePageJSON())) //nolint:errcheck
	})

	pm := &PageManager{client: client}
	page, err := pm.Create(testContext(), ParentRef{Type: "database_id", DatabaseID: "db-123"})
	require.NoError(t, err)
	assert.Equal(t, "page", page.Object)
	assert.Equal(t, "page-123", page.ID)
}

func TestPageManager_Retrieve(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		verifyAuth(t, r)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/pages/page-123", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(samplePageJSON())) //nolint:errcheck
	})

	pm := &PageManager{client: client}
	page, err := pm.Retrieve(testContext(), "page-123")
	require.NoError(t, err)
	assert.Equal(t, "page-123", page.ID)
	assert.Equal(t, "page", page.Object)
	assert.False(t, page.Archived)
}

func TestPageManager_Update(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		verifyAuth(t, r)
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/v1/pages/page-123", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(samplePageJSON())) //nolint:errcheck
	})

	pm := &PageManager{client: client}
	page, err := pm.Update(testContext(), "page-123")
	require.NoError(t, err)
	assert.Equal(t, "page-123", page.ID)
}

func TestPageManager_Trash(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		verifyAuth(t, r)
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/v1/pages/page-123", r.URL.Path)

		body := readBody(t, r)
		assert.Equal(t, true, body["in_trash"])

		// Return a page marked as in_trash.
		resp := `{
			"object": "page",
			"id": "page-123",
			"created_time": "2024-01-01T00:00:00Z",
			"created_by": {"object": "user", "id": "user-1"},
			"last_edited_time": "2024-01-02T00:00:00Z",
			"last_edited_by": {"object": "user", "id": "user-2"},
			"in_trash": true
		}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp)) //nolint:errcheck
	})

	pm := &PageManager{client: client}
	page, err := pm.Trash(testContext(), "page-123")
	require.NoError(t, err)
	assert.Equal(t, "page-123", page.ID)
	assert.True(t, page.InTrash)
}

func TestPageManager_RetrieveProperty(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		verifyAuth(t, r)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/pages/page-123/properties/prop-1", r.URL.Path)

		resp := `{
			"object": "property_item",
			"id": "prop-1",
			"type": "title",
			"title": [{"type": "text", "text": {"content": "Hello"}, "plain_text": "Hello"}]
		}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp)) //nolint:errcheck
	})

	pm := &PageManager{client: client}
	prop, err := pm.RetrieveProperty(testContext(), "page-123", "prop-1")
	require.NoError(t, err)
	assert.Equal(t, "prop-1", prop.ID)
	assert.Equal(t, PropertyType("title"), prop.Type)
}

// =============================================================================
// BlockManager tests
// =============================================================================

func TestBlockManager_Retrieve(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		verifyAuth(t, r)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/blocks/block-123", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(sampleBlockJSON())) //nolint:errcheck
	})

	bm := &BlockManager{client: client}
	block, err := bm.Retrieve(testContext(), "block-123")
	require.NoError(t, err)
	assert.Equal(t, "block", block.Object)
	assert.Equal(t, "block-123", block.ID)
	assert.Equal(t, "paragraph", block.Type)
	assert.NotNil(t, block.Paragraph)
}

func TestBlockManager_Update(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		verifyAuth(t, r)
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/v1/blocks/block-123", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(sampleBlockJSON())) //nolint:errcheck
	})

	bm := &BlockManager{client: client}
	block := &Block{Type: "paragraph", Paragraph: &RichTextBlock{RichText: []TextObject{{Text: TextItem{Content: "updated"}}}}}
	result, err := bm.Update(testContext(), "block-123", block)
	require.NoError(t, err)
	assert.Equal(t, "block-123", result.ID)
}

func TestBlockManager_Delete(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		verifyAuth(t, r)
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/v1/blocks/block-123", r.URL.Path)

		// Notion returns the deleted block in trash.
		resp := `{
			"object": "block",
			"id": "block-123",
			"type": "paragraph",
			"created_time": "2024-01-01T00:00:00Z",
			"created_by": {"object": "user", "id": "user-1"},
			"last_edited_time": "2024-01-02T00:00:00Z",
			"last_edited_by": {"object": "user", "id": "user-2"},
			"has_children": false,
			"in_trash": true,
			"paragraph": {"rich_text": []}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp)) //nolint:errcheck
	})

	bm := &BlockManager{client: client}
	block, err := bm.Delete(testContext(), "block-123")
	require.NoError(t, err)
	assert.Equal(t, "block-123", block.ID)
	assert.True(t, block.InTrash)
}

func TestBlockManager_Children_SinglePage(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		verifyAuth(t, r)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/blocks/block-parent/children", r.URL.Path)

		blocks := []map[string]any{
			{
				"object": "block", "id": "child-1", "type": "paragraph",
				"created_time": "2024-01-01T00:00:00Z",
				"created_by":   map[string]any{"object": "user", "id": "u1"},
				"last_edited_time": "2024-01-01T00:00:00Z",
				"last_edited_by":   map[string]any{"object": "user", "id": "u1"},
				"has_children":     false,
				"paragraph":        map[string]any{"rich_text": []any{}},
			},
		}
		jsonRespond(w, http.StatusOK, paginatedResponse(blocks, false, ""))
	})

	bm := &BlockManager{client: client}
	blocks, err := bm.Children(testContext(), "block-parent", nil)
	require.NoError(t, err)
	assert.Len(t, blocks, 1)
	assert.Equal(t, "child-1", blocks[0].ID)
}

func TestBlockManager_Children_Pagination(t *testing.T) {
	callCount := 0
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++

		blocks := []map[string]any{
			{
				"object": "block", "id": fmt.Sprintf("child-%d", callCount),
				"type": "paragraph", "created_time": "2024-01-01T00:00:00Z",
				"created_by":       map[string]any{"object": "user", "id": "u1"},
				"last_edited_time": "2024-01-01T00:00:00Z",
				"last_edited_by":   map[string]any{"object": "user", "id": "u1"},
				"has_children":     false,
				"paragraph":        map[string]any{"rich_text": []any{}},
			},
		}

		if callCount == 1 {
			jsonRespond(w, http.StatusOK, paginatedResponse(blocks, true, "next-cursor"))
		} else {
			jsonRespond(w, http.StatusOK, paginatedResponse(blocks, false, ""))
		}
	})

	bm := &BlockManager{client: client}
	blocks, err := bm.Children(testContext(), "block-parent", nil)
	require.NoError(t, err)
	assert.Len(t, blocks, 2)
	assert.Equal(t, 2, callCount)
}

func TestBlockManager_AppendChildren(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		verifyAuth(t, r)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/blocks/block-parent/children", r.URL.Path)

		body := readBody(t, r)
		children, ok := body["children"].([]any)
		require.True(t, ok)
		assert.Len(t, children, 1)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(sampleBlockJSON())) //nolint:errcheck
	})

	bm := &BlockManager{client: client}
	children := []Block{
		{Type: "paragraph", Paragraph: &RichTextBlock{RichText: []TextObject{{Text: TextItem{Content: "Hello"}}}}},
	}
	block, err := bm.AppendChildren(testContext(), "block-parent", children)
	require.NoError(t, err)
	assert.Equal(t, "block-123", block.ID)
}

// =============================================================================
// SearchManager tests
// =============================================================================

func TestSearchManager_Search_WithQuery(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		verifyAuth(t, r)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/search", r.URL.Path)

		body := readBody(t, r)
		assert.Equal(t, "test query", body["query"])

		results := []json.RawMessage{
			[]byte(`{"object": "page", "id": "result-1"}`),
		}
		jsonRespond(w, http.StatusOK, paginatedResponse(results, false, ""))
	})

	sm := &SearchManager{client: client}
	resp, err := sm.Search(testContext(), "test query", nil)
	require.NoError(t, err)
	assert.Len(t, resp.Results, 1)
	assert.False(t, resp.HasMore)
}

func TestSearchManager_Search_WithFilter(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		body := readBody(t, r)
		assert.Equal(t, "test", body["query"])
		filter, ok := body["filter"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "object", filter["property"])
		assert.Equal(t, "database", filter["value"])

		results := []json.RawMessage{}
		jsonRespond(w, http.StatusOK, paginatedResponse(results, false, ""))
	})

	sm := &SearchManager{client: client}
	resp, err := sm.Search(testContext(), "test", &SearchFilter{Property: "object", Value: "database"})
	require.NoError(t, err)
	assert.Empty(t, resp.Results)
}

func TestSearchManager_Search_EmptyQuery(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		body := readBody(t, r)
		// Empty query should not be included in body.
		_, hasQuery := body["query"]
		assert.False(t, hasQuery)

		results := []json.RawMessage{}
		jsonRespond(w, http.StatusOK, paginatedResponse(results, false, ""))
	})

	sm := &SearchManager{client: client}
	resp, err := sm.Search(testContext(), "", nil)
	require.NoError(t, err)
	assert.Empty(t, resp.Results)
}

// =============================================================================
// UserManager tests
// =============================================================================

func TestUserManager_Me(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		verifyAuth(t, r)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/users/me", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(sampleUserJSON())) //nolint:errcheck
	})

	um := &UserManager{client: client}
	user, err := um.Me(testContext())
	require.NoError(t, err)
	assert.Equal(t, "user", user.Object)
	assert.Equal(t, "user-123", user.ID)
	assert.Equal(t, "Test User", user.Name)
}

func TestUserManager_Retrieve(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		verifyAuth(t, r)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/users/user-123", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(sampleUserJSON())) //nolint:errcheck
	})

	um := &UserManager{client: client}
	user, err := um.Retrieve(testContext(), "user-123")
	require.NoError(t, err)
	assert.Equal(t, "user-123", user.ID)
}

func TestUserManager_List(t *testing.T) {
	callCount := 0
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		verifyAuth(t, r)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/users", r.URL.Path)

		callCount++
		users := []map[string]any{
			{"object": "user", "id": fmt.Sprintf("user-%d", callCount), "name": "User"},
		}

		if callCount == 1 {
			jsonRespond(w, http.StatusOK, paginatedResponse(users, true, "cursor-next"))
		} else {
			jsonRespond(w, http.StatusOK, paginatedResponse(users, false, ""))
		}
	})

	um := &UserManager{client: client}
	users, err := um.List(testContext())
	require.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, 2, callCount)
}

// =============================================================================
// CommentManager tests
// =============================================================================

func TestCommentManager_List(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		verifyAuth(t, r)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/comments", r.URL.Path)
		assert.Equal(t, "block-123", r.URL.Query().Get("block_id"))

		comments := []map[string]any{
			{
				"object": "comment", "id": "comment-1",
				"created_time": "2024-01-01T00:00:00Z",
				"created_by":   map[string]any{"object": "user", "id": "user-1"},
				"rich_text":    []map[string]any{{"type": "text", "text": map[string]any{"content": "hello"}}},
			},
		}
		jsonRespond(w, http.StatusOK, paginatedResponse(comments, false, ""))
	})

	cm := &CommentManager{client: client}
	comments, err := cm.List(testContext(), "block-123")
	require.NoError(t, err)
	assert.Len(t, comments, 1)
	assert.Equal(t, "comment-1", comments[0].ID)
}

func TestCommentManager_Create(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		verifyAuth(t, r)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/comments", r.URL.Path)

		body := readBody(t, r)
		require.NotNil(t, body["parent"])
		require.NotNil(t, body["rich_text"])

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(sampleCommentJSON())) //nolint:errcheck
	})

	cm := &CommentManager{client: client}
	comment, err := cm.Create(testContext(), ParentRef{Type: "page_id", PageID: "page-123"}, []TextObject{
		{Text: TextItem{Content: "Nice work!"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "comment", comment.Object)
	assert.Equal(t, "comment-123", comment.ID)
}

// =============================================================================
// Error handling tests
// =============================================================================

func TestError_Unauthorized(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		errorRespond(w, http.StatusUnauthorized, "unauthorized", "Invalid API token")
	})

	dm := &DatabaseManager{client: client}
	db, err := dm.Retrieve(testContext(), "db-123")
	require.Error(t, err)
	assert.Nil(t, db)

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.Status)
	assert.Equal(t, "unauthorized", apiErr.Code)
	assert.Equal(t, "Invalid API token", apiErr.Message)
}

func TestError_RateLimited(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"object": "error", "status": 429, "code": "rate_limited", "message": "Speed limit"}`)) //nolint:errcheck
	})

	pm := &PageManager{client: client}
	page, err := pm.Retrieve(testContext(), "page-123")
	require.Error(t, err)
	assert.Nil(t, page)
	assert.ErrorIs(t, err, ErrRateLimited)
}

func TestError_NotFound(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		errorRespond(w, http.StatusNotFound, "object_not_found", "Could not find page with ID: page-missing")
	})

	pm := &PageManager{client: client}
	page, err := pm.Retrieve(testContext(), "page-missing")
	require.Error(t, err)
	assert.Nil(t, page)

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Status)
	assert.Equal(t, "object_not_found", apiErr.Code)
}

func TestError_InternalServerError_PlainText(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error")) //nolint:errcheck
	})

	bm := &BlockManager{client: client}
	block, err := bm.Retrieve(testContext(), "block-123")
	require.Error(t, err)
	assert.Nil(t, block)
	assert.Contains(t, err.Error(), "notion api error: [500]")
}

func TestError_BadRequest(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		errorRespond(w, http.StatusBadRequest, "validation_error", "Invalid request body")
	})

	sm := &SearchManager{client: client}
	resp, err := sm.Search(testContext(), "test", nil)
	require.Error(t, err)
	assert.Nil(t, resp)

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Status)
	assert.Equal(t, "validation_error", apiErr.Code)
}

func TestError_Forbidden(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		errorRespond(w, http.StatusForbidden, "restricted_resource", "Access denied")
	})

	um := &UserManager{client: client}
	user, err := um.Retrieve(testContext(), "user-123")
	require.Error(t, err)
	assert.Nil(t, user)

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.Status)
}

// =============================================================================
// Error propagation through manager wrappers
// =============================================================================

func TestDatabaseManager_Retrieve_ErrorWrapped(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		errorRespond(w, http.StatusNotFound, "object_not_found", "Database not found")
	})

	dm := &DatabaseManager{client: client}
	_, err := dm.Retrieve(testContext(), "db-nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retrieve database db-nonexistent")
}

func TestPageManager_Create_ErrorWrapped(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		errorRespond(w, http.StatusBadRequest, "validation_error", "Invalid parent")
	})

	pm := &PageManager{client: client}
	_, err := pm.Create(testContext(), ParentRef{Type: "page_id", PageID: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create page")
}

func TestBlockManager_Delete_ErrorWrapped(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		errorRespond(w, http.StatusNotFound, "object_not_found", "Block not found")
	})

	bm := &BlockManager{client: client}
	_, err := bm.Delete(testContext(), "block-nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete block block-nonexistent")
}

func TestUserManager_Me_ErrorWrapped(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		errorRespond(w, http.StatusUnauthorized, "unauthorized", "Bad token")
	})

	um := &UserManager{client: client}
	_, err := um.Me(testContext())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retrieve current user")
}

func TestCommentManager_Create_ErrorWrapped(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		errorRespond(w, http.StatusBadRequest, "validation_error", "Missing parent")
	})

	cm := &CommentManager{client: client}
	_, err := cm.Create(testContext(), ParentRef{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create comment")
}

// =============================================================================
// Context cancellation test
// =============================================================================

func TestContext_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		// This should never be called.
		t.Error("request should not reach server after context cancellation")
	})

	pm := &PageManager{client: client}
	_, err := pm.Retrieve(ctx, "page-123")
	require.Error(t, err)
}

// =============================================================================
// Client.WithLimiter test
// =============================================================================

func TestClient_WithLimiter(t *testing.T) {
	mgr := NewClient("2022-06-28", "test-token")
	newMgr := mgr.WithLimiter(defaultLimiter())
	assert.NotNil(t, newMgr)
	assert.NotNil(t, newMgr.client)
	// Original manager should be unchanged.
	assert.NotNil(t, mgr.client)
}

// =============================================================================
// Request body verification tests
// =============================================================================

func TestPageManager_Create_WithProperties(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		body := readBody(t, r)
		require.NotNil(t, body)

		parent, ok := body["parent"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "database_id", parent["type"])
		assert.Equal(t, "db-parent", parent["database_id"])

		props, ok := body["properties"].(map[string]any)
		require.True(t, ok)
		assert.NotNil(t, props["Name"])

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(samplePageJSON())) //nolint:errcheck
	})

	pm := &PageManager{client: client}
	titleData, _ := json.Marshal([]TextObject{{Text: TextItem{Content: "Test Page"}, PlainText: "Test Page"}})
	page, err := pm.Create(
		testContext(),
		ParentRef{Type: "database_id", DatabaseID: "db-parent"},
		&Property{Name: "Name", Type: TitleProp, Title: titleData},
	)
	require.NoError(t, err)
	assert.NotNil(t, page)
}

func TestDatabaseManager_Update_WithPayload(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Read raw body since it's passed as io.Reader.
		data, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Contains(t, string(data), "Updated Title")

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(sampleDatabaseJSON())) //nolint:errcheck
	})

	dm := &DatabaseManager{client: client}
	payload := strings.NewReader(`{"title": [{"type": "text", "text": {"content": "Updated Title"}}]}`)
	db, err := dm.Update(testContext(), "db-123", payload)
	require.NoError(t, err)
	assert.NotNil(t, db)
}

func TestDatabaseManager_Update_NilPayload(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(sampleDatabaseJSON())) //nolint:errcheck
	})

	dm := &DatabaseManager{client: client}
	db, err := dm.Update(testContext(), "db-123", nil)
	require.NoError(t, err)
	assert.NotNil(t, db)
}

// =============================================================================
// Query with filter_properties query params
// =============================================================================

func TestDatabaseManager_Query_WithFilterProperties(t *testing.T) {
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		// The path should include filter_properties query param.
		assert.Contains(t, r.URL.RawQuery, "filter_properties")

		pages := []map[string]any{}
		jsonRespond(w, http.StatusOK, paginatedResponse(pages, false, ""))
	})

	dm := &DatabaseManager{client: client}
	cond := &Condition{
		FilterProperties: []string{"title"},
	}
	pages, err := dm.Query(testContext(), "db-123", cond)
	require.NoError(t, err)
	assert.Empty(t, pages)
}
