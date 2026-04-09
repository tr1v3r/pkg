package fetch

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const defaultMaxResponseBodySize int64 = 100 * 1024 * 1024 // 100MB

var (
	mu         sync.RWMutex
	httpClient = &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        5,
			MaxIdleConnsPerHost: 5,
			MaxConnsPerHost:     10,
			Proxy:               http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12, // Enforce minimum TLS 1.2
			},
		},
	}
)

// DefaultClient returns the package-level HTTP client used by all request functions.
func DefaultClient() *http.Client {
	mu.RLock()
	defer mu.RUnlock()
	return httpClient
}

// SetDefaultClient replaces the package-level HTTP client with the given one.
func SetDefaultClient(client *http.Client) {
	mu.Lock()
	defer mu.Unlock()
	httpClient = client
}

// NewInsecureClient creates an HTTP client with TLS certificate verification disabled.
// WARNING: Only use in testing or development environments.
func NewInsecureClient() *http.Client {
	return &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        5,
			MaxIdleConnsPerHost: 5,
			MaxConnsPerHost:     10,
			Proxy:               http.ProxyFromEnvironment,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // InsecureSkipVerify is intentional for testing/development
		},
	}
}

// Get sends a GET request and returns the response body.
func Get(url string, opts ...RequestOption) ([]byte, error) {
	_, content, _, err := DoRequestWithOptions(http.MethodGet, url, opts, nil)
	return content, err
}

// CtxGet sends a GET request with context and returns the response body.
func CtxGet(ctx context.Context, url string, opts ...RequestOption) ([]byte, error) {
	_, content, _, err := DoRequestWithOptions(http.MethodGet, url, append([]RequestOption{WithContext(ctx)}, opts...), nil)
	return content, err
}

// Post sends a POST request and returns the response body.
func Post(url string, body io.Reader, opts ...RequestOption) ([]byte, error) {
	_, content, _, err := DoRequestWithOptions(http.MethodPost, url, opts, body)
	return content, err
}

// CtxPost sends a POST request with context and returns the response body.
func CtxPost(ctx context.Context, url string, body io.Reader, opts ...RequestOption) ([]byte, error) {
	_, content, _, err := DoRequestWithOptions(http.MethodPost, url, append([]RequestOption{WithContext(ctx)}, opts...), body)
	return content, err
}

// Patch sends a PATCH request and returns the response body.
func Patch(url string, body io.Reader, opts ...RequestOption) ([]byte, error) {
	_, content, _, err := DoRequestWithOptions(http.MethodPatch, url, opts, body)
	return content, err
}

// CtxPatch sends a PATCH request with context and returns the response body.
func CtxPatch(ctx context.Context, url string, body io.Reader, opts ...RequestOption) ([]byte, error) {
	_, content, _, err := DoRequestWithOptions(http.MethodPatch, url, append([]RequestOption{WithContext(ctx)}, opts...), body)
	return content, err
}

// DoRequest sends an HTTP request and returns the status code and response body.
func DoRequest(method string, url string, body io.Reader) (statusCode int, content []byte, err error) {
	statusCode, content, _, err = DoRequestWithOptions(method, url, nil, body)
	return
}

// DoRequestWithContext sends an HTTP request with context and options, returning the status code and response body.
func DoRequestWithContext(ctx context.Context, method string, url string, opts []RequestOption, body io.Reader) (
	statusCode int, content []byte, err error) {
	statusCode, content, _, err = DoRequestWithOptions(method, url, append(opts, WithContext(ctx)), body)
	return
}

// DoRequestWithOptions sends an HTTP request with options and returns the status code, response body, and headers.
func DoRequestWithOptions(method string, url string, opts []RequestOption, body io.Reader) (
	statusCode int, content []byte, respHeaders http.Header, err error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("build new request fail: %w", err)
	}

	for _, opt := range opts {
		req = opt(req)
	}

	// Extract and defer timeout cancel to prevent timer leaks from WithTimeout
	if cancel, ok := req.Context().Value(timeoutCancelKey).(context.CancelFunc); ok {
		defer cancel()
	}

	// Extract response body size limit
	var maxBodySize int64 = defaultMaxResponseBodySize
	if s, ok := req.Context().Value(maxBodySizeKey).(int64); ok {
		if s < 0 {
			maxBodySize = 0 // 0 means no limit
		} else if s > 0 {
			maxBodySize = s
		}
	}

	// Core request execution
	doRequest := func() (int, []byte, http.Header, error) {
		resp, err := DefaultClient().Do(req)
		if err != nil {
			return -1, nil, nil, err
		}
		defer resp.Body.Close() // nolint

		var reader io.Reader = resp.Body
		if maxBodySize > 0 {
			reader = io.LimitReader(resp.Body, maxBodySize)
		}
		content, err := io.ReadAll(reader)
		if err != nil {
			return -1, nil, nil, err
		}
		return resp.StatusCode, content, resp.Header, nil
	}

	// Apply middleware chain if present
	if middlewares, _ := req.Context().Value(middlewareKey).([]Middleware); len(middlewares) > 0 {
		chain := ChainMiddleware(middlewares...)
		return chain(doRequest)
	}

	return doRequest()
}
