package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"strconv"

	"golang.org/x/time/rate"

	"github.com/tr1v3r/pkg/log"
)

// httpDoer abstracts HTTP request execution for testability.
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// notionClient handles HTTP communication with the Notion API.
type notionClient struct {
	version string
	token   string
	limiter *rate.Limiter
	http    httpDoer
}

func newNotionClient(version, token string, limiter *rate.Limiter) *notionClient {
	return &notionClient{
		version: version,
		token:   token,
		limiter: limiter,
		http:    http.DefaultClient,
	}
}

func (c *notionClient) headers() map[string]string {
	return map[string]string{
		"Notion-Version": c.version,
		"Authorization":  "Bearer " + c.token,
		"Content-Type":   "application/json",
	}
}

// do performs a rate-limited HTTP request against the Notion API.
func (c *notionClient) do(ctx context.Context, method, path string, body, result any) error {
	if err := c.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter: %w", err)
	}

	endpoint := notionAPI() + path

	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	for k, v := range c.headers() {
		req.Header.Set(k, v)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request %s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return ErrRateLimited
	}

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if jsonErr := json.Unmarshal(respBody, &apiErr); jsonErr == nil && apiErr.Code != "" {
			apiErr.Status = resp.StatusCode
			return &apiErr
		}
		return fmt.Errorf("notion api error: [%d] %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}
	return nil
}

// get performs a GET request.
func (c *notionClient) get(ctx context.Context, path string, result any) error {
	return c.do(ctx, http.MethodGet, path, nil, result)
}

// post performs a POST request.
func (c *notionClient) post(ctx context.Context, path string, body, result any) error {
	return c.do(ctx, http.MethodPost, path, body, result)
}

// patch performs a PATCH request.
func (c *notionClient) patch(ctx context.Context, path string, body, result any) error {
	return c.do(ctx, http.MethodPatch, path, body, result)
}

// delete performs a DELETE request.
func (c *notionClient) delete(ctx context.Context, path string, result any) error {
	return c.do(ctx, http.MethodDelete, path, nil, result)
}

// paginateEach lazily iterates over a paginated endpoint, yielding one item at a time.
//
// Pagination parameters travel per HTTP method, mirroring the Notion API contract:
// POST endpoints (e.g. database query) read them from the JSON body, while GET
// endpoints (users, block children, comments) read them exclusively from URL query
// parameters and ignore request bodies. Sending ListOptions as a GET body therefore
// left the cursor stuck on the first page, looping forever over duplicate items, so
// GET requests are converted to query parameters before dispatch.
func paginateEach[T any](ctx context.Context, c *notionClient, method, path string,
	bodyFn func(cursor string) any) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var nextCursor string
		for {
			body := bodyFn(nextCursor)
			reqPath, reqBody := path, body
			if method == http.MethodGet {
				opts, ok := body.(*ListOptions)
				if !ok {
					var zero T
					yield(zero, fmt.Errorf("paginate %s %s: GET pagination requires ListOptions, got %T", method, path, body))
					return
				}
				reqPath, reqBody = withListQuery(path, opts), nil
			}
			var resp ListResponse[T]
			if err := c.do(ctx, method, reqPath, reqBody, &resp); err != nil {
				var zero T
				yield(zero, err)
				return
			}
			for _, item := range resp.Results {
				if !yield(item, nil) {
					return
				}
			}
			if !resp.HasMore {
				return
			}
			nextCursor = resp.NextCursor
			log.CtxDebugf(ctx, "paginated fetch: fetching next page with cursor %s", nextCursor)
		}
	}
}

// paginateAll fetches all pages of a paginated endpoint.
func paginateAll[T any](ctx context.Context, c *notionClient, method, path string,
	bodyFn func(cursor string) any) ([]T, error) {
	var all []T
	for item, err := range paginateEach[T](ctx, c, method, path, bodyFn) {
		if err != nil {
			return nil, err
		}
		all = append(all, item)
	}
	return all, nil
}

// withListQuery merges pagination options into the query string of path, preserving
// any parameters already present (e.g. block_id when listing comments).
func withListQuery(path string, opts *ListOptions) string {
	u, err := url.Parse(path)
	if err != nil {
		return path
	}
	q := u.Query()
	if opts.PageSize > 0 {
		q.Set("page_size", strconv.Itoa(opts.PageSize))
	}
	if opts.Cursor != "" {
		q.Set("start_cursor", opts.Cursor)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
