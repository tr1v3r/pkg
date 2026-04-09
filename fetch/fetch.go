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

// DefaultClient return default client
func DefaultClient() *http.Client {
	mu.RLock()
	defer mu.RUnlock()
	return httpClient
}

// SetDefaultClient set client replace default client
func SetDefaultClient(client *http.Client) {
	mu.Lock()
	defer mu.Unlock()
	httpClient = client
}

// NewInsecureClient creates a new HTTP client with disabled certificate verification
// WARNING: Only use for testing or development environments
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

// Get ...
func Get(url string, opts ...RequestOption) ([]byte, error) {
	_, content, _, err := DoRequestWithOptions(http.MethodGet, url, opts, nil)
	return content, err
}

// CtxGet ...
func CtxGet(ctx context.Context, url string, opts ...RequestOption) ([]byte, error) {
	_, content, _, err := DoRequestWithOptions(http.MethodGet, url, append([]RequestOption{WithContext(ctx)}, opts...), nil)
	return content, err
}

// Post ...
func Post(url string, body io.Reader, opts ...RequestOption) ([]byte, error) {
	_, content, _, err := DoRequestWithOptions(http.MethodPost, url, opts, body)
	return content, err
}

// CtxPost ...
func CtxPost(ctx context.Context, url string, body io.Reader, opts ...RequestOption) ([]byte, error) {
	_, content, _, err := DoRequestWithOptions(http.MethodPost, url, append([]RequestOption{WithContext(ctx)}, opts...), body)
	return content, err
}

// Patch ...
func Patch(url string, body io.Reader, opts ...RequestOption) ([]byte, error) {
	_, content, _, err := DoRequestWithOptions(http.MethodPatch, url, opts, body)
	return content, err
}

// CtxPatch ...
func CtxPatch(ctx context.Context, url string, body io.Reader, opts ...RequestOption) ([]byte, error) {
	_, content, _, err := DoRequestWithOptions(http.MethodPatch, url, append([]RequestOption{WithContext(ctx)}, opts...), body)
	return content, err
}

// DoRequest 进行HTTP请求
func DoRequest(method string, url string, body io.Reader) (statusCode int, content []byte, err error) {
	statusCode, content, _, err = DoRequestWithOptions(method, url, nil, body)
	return
}

// DoRequestWithContext 进行HTTP请求
func DoRequestWithContext(ctx context.Context, method string, url string, opts []RequestOption, body io.Reader) (
	statusCode int, content []byte, err error) {
	statusCode, content, _, err = DoRequestWithOptions(method, url, append(opts, WithContext(ctx)), body)
	return
}

// DoRequestWithOptions 进行HTTP请求并返回响应头
func DoRequestWithOptions(method string, url string, opts []RequestOption, body io.Reader) (
	statusCode int, content []byte, respHeaders http.Header, err error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("build new request fail: %w", err)
	}

	var maxBodySize int64 = defaultMaxResponseBodySize
	for _, opt := range opts {
		req = opt(req)
	}
	if s, ok := req.Context().Value(maxBodySizeKey).(int64); ok {
		if s < 0 {
			maxBodySize = 0 // 0 means no limit for io.LimitReader
		} else if s > 0 {
			maxBodySize = s
		}
	}

	resp, err := DefaultClient().Do(req)
	if err != nil {
		return -1, nil, nil, err
	}
	defer resp.Body.Close() // nolint

	var reader io.Reader = resp.Body
	if maxBodySize > 0 {
		reader = io.LimitReader(resp.Body, maxBodySize)
	}
	content, err = io.ReadAll(reader)
	if err != nil {
		return -1, nil, nil, err
	}
	return resp.StatusCode, content, resp.Header, nil
}
