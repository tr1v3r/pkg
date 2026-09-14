package notion

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// GET pagination regression (BUG-AUDIT-REPORT H2)
//
// The real Notion API reads pagination on GET list endpoints exclusively from URL
// query parameters (start_cursor / page_size) and ignores JSON request bodies. The
// mocks here are spec-faithful: pages are keyed by the incoming query cursor, and a
// request cap turns a cursor that fails to advance into a fast 400 failure instead
// of an infinite pagination loop.
// =============================================================================

// specPage is one page of a spec-faithful GET list mock.
type specPage struct {
	items      []map[string]any
	hasMore    bool
	nextCursor string
}

// specGETRecorder captures what the spec-faithful mock observed per request.
type specGETRecorder struct {
	calls     int
	cursors   []string
	pageSizes []string
	paths     []string
	queries   []string
	bodies    []string
}

// specGETListServer wires a mock Notion server for a GET list endpoint. Like the real
// API it reads pagination only from query parameters and ignores the body entirely.
// pages maps the incoming start_cursor to the page served for it; the "" key is the
// first page. Once requests exceed len(pages) — meaning the client's cursor never
// advanced — the handler answers 400 so a broken client errors out fast.
func specGETListServer(t *testing.T, pages map[string]specPage) (*notionClient, *specGETRecorder) {
	t.Helper()
	rec := &specGETRecorder{}
	_, client := mockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)

		rec.calls++
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		rec.bodies = append(rec.bodies, string(body))
		rec.cursors = append(rec.cursors, r.URL.Query().Get("start_cursor"))
		rec.pageSizes = append(rec.pageSizes, r.URL.Query().Get("page_size"))
		rec.paths = append(rec.paths, r.URL.Path)
		rec.queries = append(rec.queries, r.URL.RawQuery)

		if rec.calls > len(pages) {
			errorRespond(w, http.StatusBadRequest, "validation_error",
				"pagination did not advance: too many requests for this cursor chain")
			return
		}
		page, ok := pages[r.URL.Query().Get("start_cursor")]
		if !ok {
			errorRespond(w, http.StatusBadRequest, "validation_error",
				fmt.Sprintf("unknown start_cursor %q", r.URL.Query().Get("start_cursor")))
			return
		}
		jsonRespond(w, http.StatusOK, paginatedResponse(page.items, page.hasMore, page.nextCursor))
	})
	return client, rec
}

// userItem, blockItem and commentItem build minimal fixtures for the generic list
// envelope, mirroring the shapes the existing API tests already use.
func userItem(id string) map[string]any {
	return map[string]any{"object": "user", "id": id, "name": "User " + id}
}

func blockItem(id string) map[string]any {
	return map[string]any{
		"object": "block", "id": id, "type": "paragraph",
		"created_time":     "2024-01-01T00:00:00Z",
		"created_by":       map[string]any{"object": "user", "id": "u1"},
		"last_edited_time": "2024-01-01T00:00:00Z",
		"last_edited_by":   map[string]any{"object": "user", "id": "u1"},
		"has_children":     false,
		"paragraph":        map[string]any{"rich_text": []any{}},
	}
}

func commentItem(id string) map[string]any {
	return map[string]any{
		"object": "comment", "id": id,
		"created_time": "2024-01-01T00:00:00Z",
		"created_by":   map[string]any{"object": "user", "id": "user-1"},
		"rich_text":    []map[string]any{{"type": "text", "text": map[string]any{"content": "hello"}}},
	}
}

// TestGETListPagination_QueryParams walks every GET list endpoint through a
// three-page cursor chain against the spec-faithful mock and asserts that the
// cursor advances page by page via query parameters, results arrive complete and
// duplicate-free, and the iteration converges in exactly len(pages) requests.
func TestGETListPagination_QueryParams(t *testing.T) {
	tests := []struct {
		name        string
		item        func(id string) map[string]any
		call        func(t *testing.T, nc *notionClient) ([]string, error)
		wantPath    string
		wantExtra   map[string]string
		wantPageSz  string
		wantMaxCall int
	}{
		{
			name: "users list",
			item: userItem,
			call: func(t *testing.T, nc *notionClient) ([]string, error) {
				um := &UserManager{client: nc}
				users, err := um.List(testContext())
				if err != nil {
					return nil, err
				}
				ids := make([]string, 0, len(users))
				for _, u := range users {
					ids = append(ids, u.ID)
				}
				return ids, nil
			},
			wantPath:   "/v1/users",
			wantPageSz: "100",
		},
		{
			name: "block children",
			item: blockItem,
			call: func(t *testing.T, nc *notionClient) ([]string, error) {
				bm := &BlockManager{client: nc}
				blocks, err := bm.Children(testContext(), "block-parent", &ListOptions{PageSize: 2})
				if err != nil {
					return nil, err
				}
				ids := make([]string, 0, len(blocks))
				for _, b := range blocks {
					ids = append(ids, b.ID)
				}
				return ids, nil
			},
			wantPath:   "/v1/blocks/block-parent/children",
			wantPageSz: "2",
		},
		{
			name: "comments list",
			item: commentItem,
			call: func(t *testing.T, nc *notionClient) ([]string, error) {
				cm := &CommentManager{client: nc}
				comments, err := cm.List(testContext(), "block-123")
				if err != nil {
					return nil, err
				}
				ids := make([]string, 0, len(comments))
				for _, c := range comments {
					ids = append(ids, c.ID)
				}
				return ids, nil
			},
			wantPath:    "/v1/comments",
			wantExtra:   map[string]string{"block_id": "block-123"},
			wantPageSz:  "100",
			wantMaxCall: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pages := map[string]specPage{
				"": {
					items:      []map[string]any{tt.item("id-1"), tt.item("id-2")},
					hasMore:    true,
					nextCursor: "cursor-1",
				},
				"cursor-1": {
					items:      []map[string]any{tt.item("id-3"), tt.item("id-4")},
					hasMore:    true,
					nextCursor: "cursor-2",
				},
				"cursor-2": {
					items: []map[string]any{tt.item("id-5"), tt.item("id-6")},
				},
			}

			client, rec := specGETListServer(t, pages)
			ids, err := tt.call(t, client)
			require.NoError(t, err)

			// Cursor advanced page by page and iteration converged: one request per page.
			assert.Equal(t, []string{"", "cursor-1", "cursor-2"}, rec.cursors)
			assert.Equal(t, len(pages), rec.calls)
			if tt.wantMaxCall > 0 {
				assert.LessOrEqual(t, rec.calls, tt.wantMaxCall)
			}

			// All items aggregated once, in page order — no duplicates, none missing.
			assert.Equal(t, []string{"id-1", "id-2", "id-3", "id-4", "id-5", "id-6"}, ids)

			// Pagination never rides in a JSON body on GET endpoints.
			for _, b := range rec.bodies {
				assert.Empty(t, b, "GET pagination must not be sent as a request body")
			}
			// page_size travels in the query on every request.
			for _, ps := range rec.pageSizes {
				assert.Equal(t, tt.wantPageSz, ps)
			}
			for _, p := range rec.paths {
				assert.Equal(t, tt.wantPath, p)
			}
			// Endpoint-specific query parameters survive the pagination merge.
			for _, q := range rec.queries {
				parsed, perr := url.ParseQuery(q)
				require.NoError(t, perr)
				for k, want := range tt.wantExtra {
					assert.Equal(t, want, parsed.Get(k))
				}
			}
		})
	}
}

// TestWithListQuery pins how pagination options merge into a request path.
func TestWithListQuery(t *testing.T) {
	tests := []struct {
		name string
		path string
		opts *ListOptions
		want string
	}{
		{
			name: "empty options leave path untouched",
			path: "/users",
			opts: &ListOptions{},
			want: "/users",
		},
		{
			name: "page size only",
			path: "/users",
			opts: &ListOptions{PageSize: 100},
			want: "/users?page_size=100",
		},
		{
			name: "cursor only",
			path: "/blocks/b1/children",
			opts: &ListOptions{Cursor: "cur-1"},
			want: "/blocks/b1/children?start_cursor=cur-1",
		},
		{
			name: "cursor is url-encoded",
			path: "/users",
			opts: &ListOptions{Cursor: "a b&c"},
			want: "/users?start_cursor=a+b%26c",
		},
		{
			name: "merges with existing query preserving other params",
			path: "/comments?block_id=block-123",
			opts: &ListOptions{PageSize: 100, Cursor: "cur-1"},
			want: "/comments?block_id=block-123&page_size=100&start_cursor=cur-1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, withListQuery(tt.path, tt.opts))
		})
	}
}
