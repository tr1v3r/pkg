package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/time/rate"
)

// mockServer creates an httptest.Server that delegates to handler and returns
// the server (for cleanup) and a notionClient wired to use it.
func mockServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *notionClient) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	client := &http.Client{
		Transport: http.DefaultTransport,
	}
	// Override the notionAPI base URL via a client wrapper that rewrites URLs.
	nc := &notionClient{
		version: "2022-06-28",
		token:   "test-token",
		limiter: rate.NewLimiter(1000, 1000),
		http:    urlRewriteClient{base: srv.URL, inner: client},
	}
	return srv, nc
}

// urlRewriteClient is an httpDoer that rewrites request URLs to point at a
// test server while preserving the path and query string.
type urlRewriteClient struct {
	base  string
	inner *http.Client
}

func (r urlRewriteClient) Do(req *http.Request) (*http.Response, error) {
	// Replace scheme+host with test server, keep path+query.
	baseURL, err := parseBaseURL(r.base)
	if err != nil {
		return nil, err
	}
	req.URL.Host = baseURL.Host
	req.URL.Scheme = baseURL.Scheme
	req.Host = baseURL.Host
	return r.inner.Do(req)
}

// baseURL holds parsed scheme and host from the test server URL.
type baseURL struct {
	Scheme string
	Host   string
}

func parseBaseURL(raw string) (baseURL, error) {
	// Simple parse: expect "http://host:port".
	for _, scheme := range []string{"http://", "https://"} {
		if len(raw) > len(scheme) && raw[:len(scheme)] == scheme {
			return baseURL{Scheme: scheme[:len(scheme)-3], Host: raw[len(scheme):]}, nil
		}
	}
	return baseURL{}, fmt.Errorf("invalid base URL: %s", raw)
}

// testContext returns a background context for tests.
func testContext() context.Context {
	return context.Background()
}

// --- Response helpers ---

// jsonRespond writes a JSON response with the given status code.
func jsonRespond(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	data, _ := json.Marshal(v)
	w.Write(data) //nolint:errcheck
}

// errorRespond writes a Notion-style API error response.
func errorRespond(w http.ResponseWriter, statusCode int, code, message string) {
	jsonRespond(w, statusCode, map[string]any{
		"object":  "error",
		"status":  statusCode,
		"code":    code,
		"message": message,
	})
}

// readBody reads and returns the request body as a map for assertion.
func readBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read request body: %v", err)
	}
	if len(data) == 0 {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	return m
}

// verifyAuth checks that required Notion headers are present.
func verifyAuth(t *testing.T, r *http.Request) {
	t.Helper()
	if v := r.Header.Get("Notion-Version"); v != "2022-06-28" {
		t.Errorf("expected Notion-Version header '2022-06-28', got %q", v)
	}
	if v := r.Header.Get("Authorization"); v != "Bearer test-token" {
		t.Errorf("expected Authorization header 'Bearer test-token', got %q", v)
	}
	if v := r.Header.Get("Content-Type"); v != "application/json" {
		t.Errorf("expected Content-Type header 'application/json', got %q", v)
	}
}

// --- Fixture data helpers -----

// sampleDatabaseJSON returns a JSON-encoded Notion database response.
func sampleDatabaseJSON() string {
	return `{
		"object": "database",
		"id": "db-123",
		"created_time": "2024-01-01T00:00:00Z",
		"created_by": {"object": "user", "id": "user-1"},
		"last_edited_time": "2024-01-02T00:00:00Z",
		"last_edited_by": {"object": "user", "id": "user-2"},
		"title": [{"type": "text", "text": {"content": "Test DB"}, "plain_text": "Test DB"}],
		"url": "https://notion.so/db-123",
		"properties": {
			"Name": {"id": "prop-1", "name": "Name", "type": "title", "title": []}
		}
	}`
}

// samplePageJSON returns a JSON-encoded Notion page response.
func samplePageJSON() string {
	return `{
		"object": "page",
		"id": "page-123",
		"created_time": "2024-01-01T00:00:00Z",
		"created_by": {"object": "user", "id": "user-1"},
		"last_edited_time": "2024-01-02T00:00:00Z",
		"last_edited_by": {"object": "user", "id": "user-2"},
		"parent": {"type": "database_id", "database_id": "db-123"},
		"url": "https://notion.so/page-123",
		"archived": false,
		"in_trash": false,
		"properties": {
			"Name": {"id": "prop-1", "name": "Name", "type": "title", "title": [{"type": "text", "text": {"content": "Hello"}, "plain_text": "Hello"}]}
		}
	}`
}

// sampleBlockJSON returns a JSON-encoded Notion block response.
func sampleBlockJSON() string {
	return `{
		"object": "block",
		"id": "block-123",
		"parent": {"type": "page_id", "page_id": "page-123"},
		"type": "paragraph",
		"created_time": "2024-01-01T00:00:00Z",
		"created_by": {"object": "user", "id": "user-1"},
		"last_edited_time": "2024-01-02T00:00:00Z",
		"last_edited_by": {"object": "user", "id": "user-2"},
		"has_children": false,
		"paragraph": {"rich_text": [{"type": "text", "text": {"content": "Hello world"}, "plain_text": "Hello world"}]}
	}`
}

// sampleUserJSON returns a JSON-encoded Notion user response.
func sampleUserJSON() string {
	return `{
		"object": "user",
		"id": "user-123",
		"type": "person",
		"name": "Test User",
		"avatar_url": "https://example.com/avatar.png"
	}`
}

// sampleCommentJSON returns a JSON-encoded Notion comment response.
func sampleCommentJSON() string {
	return `{
		"object": "comment",
		"id": "comment-123",
		"parent": {"type": "page_id", "page_id": "page-123"},
		"discussion_id": "disc-123",
		"created_time": "2024-01-01T00:00:00Z",
		"last_edited_time": "2024-01-01T00:00:00Z",
		"created_by": {"object": "user", "id": "user-1"},
		"rich_text": [{"type": "text", "text": {"content": "Nice work!"}, "plain_text": "Nice work!"}]
	}`
}

// paginatedResponse wraps results in a paginated response envelope.
func paginatedResponse(results any, hasMore bool, nextCursor string) map[string]any {
	return map[string]any{
		"object":     "list",
		"results":    results,
		"has_more":   hasMore,
		"next_cursor": nextCursor,
	}
}
