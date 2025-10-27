package fetch

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestWithHeader(t *testing.T) {
	const (
		headerValue1 = "value1"
		headerValue2 = "value2"
	)
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	opt := WithHeader("X-Test", headerValue1, headerValue2)
	result := opt(req)

	if result.Header.Get("X-Test") != headerValue1 {
		t.Error("WithHeader did not set first value correctly")
	}

	values := result.Header.Values("X-Test")
	if len(values) != 2 {
		t.Errorf("expected 2 header values, got %d", len(values))
	}
	if values[0] != headerValue1 || values[1] != headerValue2 {
		t.Errorf("header values incorrect: %v", values)
	}
}

func TestWithHeaders(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	headers := map[string][]string{
		"X-Test1": {"value1"},
		"X-Test2": {"value2", "value3"},
	}

	opt := WithHeaders(headers)
	result := opt(req)

	if result.Header.Get("X-Test1") != "value1" {
		t.Error("WithHeaders did not set X-Test1 correctly")
	}

	values := result.Header.Values("X-Test2")
	if len(values) != 2 {
		t.Errorf("expected 2 values for X-Test2, got %d", len(values))
	}
	if values[0] != "value2" || values[1] != "value3" {
		t.Errorf("X-Test2 values incorrect: %v", values)
	}
}

func TestWithSetHeader(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.Header.Set("X-Test", "old-value")

	opt := WithSetHeader("X-Test", "new-value")
	result := opt(req)

	if result.Header.Get("X-Test") != "new-value" {
		t.Error("WithSetHeader did not override existing header")
	}

	values := result.Header.Values("X-Test")
	if len(values) != 1 {
		t.Errorf("expected 1 header value after Set, got %d", len(values))
	}
}

func TestWithoutHeader(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.Header.Set("X-Test", "value")

	opt := WithoutHeader("X-Test")
	result := opt(req)

	if result.Header.Get("X-Test") != "" {
		t.Error("WithoutHeader did not remove header")
	}
}

func TestWithContentType(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	opt := WithContentType("application/xml")
	result := opt(req)

	if result.Header.Get("Content-Type") != "application/xml" {
		t.Error("WithContentType did not set Content-Type header")
	}
}

func TestWithContentTypeJSON(t *testing.T) {
	const contentTypeJSON = "application/json"
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	opt := WithContentTypeJSON()
	result := opt(req)

	if result.Header.Get("Content-Type") != contentTypeJSON {
		t.Error("WithContentTypeJSON did not set Content-Type to application/json")
	}
}

func TestWithAuthToken(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	opt := WithAuthToken("Bearer token123")
	result := opt(req)

	if result.Header.Get("Authorization") != "Bearer token123" {
		t.Error("WithAuthToken did not set Authorization header")
	}
}

func TestWithContext(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	type contextKey string
	ctx := context.WithValue(context.Background(), contextKey("test-key"), "test-value")

	opt := WithContext(ctx)
	result := opt(req)

	if result.Context().Value("test-key") != "test-value" {
		t.Error("WithContext did not set the request context")
	}
}

func TestWithUserAgent(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	opt := WithUserAgent("TestAgent/1.0")
	result := opt(req)

	if result.Header.Get("User-Agent") != "TestAgent/1.0" {
		t.Error("WithUserAgent did not set User-Agent header")
	}
}

func TestWithBasicAuth(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	opt := WithBasicAuth("username", "password")
	result := opt(req)

	username, password, ok := result.BasicAuth()
	if !ok {
		t.Error("WithBasicAuth did not set basic auth")
	}
	if username != "username" || password != "password" {
		t.Errorf("basic auth credentials incorrect: %s:%s", username, password)
	}
}

func TestWithTimeout(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	opt := WithTimeout(5 * time.Second)
	result := opt(req)

	deadline, ok := result.Context().Deadline()
	if !ok {
		t.Error("WithTimeout did not set a deadline")
	}

	expectedDeadline := time.Now().Add(5 * time.Second)
	if deadline.Sub(expectedDeadline) > 100*time.Millisecond {
		t.Errorf("deadline is too far from expected: %v vs %v", deadline, expectedDeadline)
	}
}

func TestWithQueryParams(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	params := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	opt := WithQueryParams(params)
	result := opt(req)

	query := result.URL.Query()
	if query.Get("key1") != "value1" {
		t.Error("WithQueryParams did not set key1")
	}
	if query.Get("key2") != "value2" {
		t.Error("WithQueryParams did not set key2")
	}
}

func TestMultipleOptions(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	opts := []RequestOption{
		WithContentTypeJSON(),
		WithUserAgent("TestAgent"),
		WithAuthToken("Bearer token"),
	}

	result := req
	for _, opt := range opts {
		result = opt(result)
	}

	if result.Header.Get("Content-Type") != "application/json" {
		t.Error("Content-Type not set correctly with multiple options")
	}
	if result.Header.Get("User-Agent") != "TestAgent" {
		t.Error("User-Agent not set correctly with multiple options")
	}
	if result.Header.Get("Authorization") != "Bearer token" {
		t.Error("Authorization not set correctly with multiple options")
	}
}

func TestOptionChaining(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	// Chain options together
	result := WithContentTypeJSON()(
		WithUserAgent("TestAgent")(
			WithAuthToken("Bearer token")(req),
		),
	)

	if result.Header.Get("Content-Type") != "application/json" {
		t.Error("Content-Type not set correctly with chained options")
	}
	if result.Header.Get("User-Agent") != "TestAgent" {
		t.Error("User-Agent not set correctly with chained options")
	}
	if result.Header.Get("Authorization") != "Bearer token" {
		t.Error("Authorization not set correctly with chained options")
	}
}

func TestWithQueryParamsPreservesExisting(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com?existing=value", nil)

	params := map[string]string{
		"new": "newvalue",
	}

	opt := WithQueryParams(params)
	result := opt(req)

	query := result.URL.Query()
	if query.Get("existing") != "value" {
		t.Error("WithQueryParams removed existing query parameters")
	}
	if query.Get("new") != "newvalue" {
		t.Error("WithQueryParams did not add new query parameters")
	}
}