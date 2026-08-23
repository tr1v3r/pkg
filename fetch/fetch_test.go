package fetch

import (
	"net/http"
	"strings"
	"testing"
)

func TestGet(t *testing.T) {
	srv := newTestBackend(t)

	data, err := Get(srv.URL+"/json", WithContentTypeJSON())
	if err != nil {
		t.Fatalf("Get fail: %s", err)
	}
	if !strings.Contains(string(data), "sample") {
		t.Errorf("Get body = %s, want it to contain %q", data, "sample")
	}
}

func TestGetWithRetry(t *testing.T) {
	srv := newTestBackend(t)

	statusCode, _, _, err := DoRequestWithRetry(
		"GET",
		srv.URL+"/status/500",
		[]RequestOption{WithContentTypeJSON()},
		nil,
		WithMaxAttempts(2),
	)

	// After retries are exhausted the helper reports a retryable error.
	if err == nil {
		t.Fatal("expected retry error for persistent 500, got nil")
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("error = %v, want it to mention HTTP 500", err)
	}
	if statusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", statusCode, http.StatusInternalServerError)
	}
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
