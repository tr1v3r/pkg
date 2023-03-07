package fetch

import (
	"crypto/tls"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

var (
	mu         sync.RWMutex
	httpClient = &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        5,
			MaxIdleConnsPerHost: 5,
			MaxConnsPerHost:     10,
			Proxy:               http.ProxyFromEnvironment,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
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

// Get ...
func Get(url string) ([]byte, error) { return GetWithHeaders(url) }

// GetWithToken ...
func GetWithToken(url string, token string) ([]byte, error) {
	return GetWithHeaders(url, WithHeader("Authorization", token))
}

// GetWithHeaders ...
func GetWithHeaders(url string, opts ...RequestOption) ([]byte, error) {
	_, content, _, err := DoRequestWithOptions("GET", url, opts, nil)
	return content, err
}

// Post ...
func Post(url string, body io.Reader) ([]byte, error) {
	return PostWithHeaders(url, body, nil)
}

func PostWithHeaders(url string, body io.Reader, opts ...RequestOption) ([]byte, error) {
	_, content, _, err := DoRequestWithOptions("POST", url, opts, body)
	return content, err
}

// DoRequest 进行HTTP请求
func DoRequest(method string, url string, body io.Reader) (statusCode int, content []byte, err error) {
	statusCode, content, _, err = DoRequestWithOptions(method, url, nil, body)
	return
}

// DoRequestWithOptions 进行HTTP请求并返回响应头
func DoRequestWithOptions(method string, url string, opts []RequestOption, body io.Reader) (statusCode int, content []byte, respHeaders http.Header, err error) {
	req, _ := http.NewRequest(method, url, body)
	for _, opt := range opts {
		req = opt(req)
	}

	resp, err := DefaultClient().Do(req)
	if err != nil {
		return -1, nil, nil, err
	}
	defer resp.Body.Close() // nolint

	content, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, nil, nil, err
	}
	return resp.StatusCode, content, resp.Header, nil
}