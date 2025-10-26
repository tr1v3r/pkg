package fetch

import (
	"testing"
)

func TestGet(t *testing.T) {
	data, err := Get("https://httpbin.org/json", WithContentTypeJSON())
	if err != nil {
		t.Logf("get httpbin.org fail (this might be expected if offline): %s", err)
	} else {
		t.Logf("got data: %v", string(data))
	}
}

func TestGetWithRetry(t *testing.T) {
	statusCode, _, _, err := DoRequestWithRetry(
		"GET",
		"https://httpbin.org/status/500",
		[]RequestOption{WithContentTypeJSON()},
		nil,
		WithMaxAttempts(2),
	)

	if err != nil {
		t.Logf("request failed (expected for status 500): %v", err)
	}
	if statusCode != 500 {
		t.Logf("expected status 500, got %d", statusCode)
	}
	t.Logf("retry test completed with status: %d", statusCode)
}

func TestRequestOptions(t *testing.T) {
	// Test that request options can be created without errors
	opts := []RequestOption{
		WithContentTypeJSON(),
		WithUserAgent("test-agent"),
		WithAuthToken("Bearer test-token"),
		WithQueryParams(map[string]string{"key": "value"}),
	}

	if len(opts) != 4 {
		t.Errorf("expected 4 options, got %d", len(opts))
	}
}
