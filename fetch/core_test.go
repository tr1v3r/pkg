package fetch

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestDefaultClient(t *testing.T) {
	client := DefaultClient()
	if client == nil {
		t.Error("DefaultClient() returned nil")
		return
	}
	if client.Timeout != 60*time.Second {
		t.Errorf("expected timeout 60s, got %v", client.Timeout)
	}
}

func TestSetDefaultClient(t *testing.T) {
	originalClient := DefaultClient()

	// Create a new client with different timeout
	newClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	SetDefaultClient(newClient)

	currentClient := DefaultClient()
	if currentClient.Timeout != 30*time.Second {
		t.Errorf("expected timeout 30s after SetDefaultClient, got %v", currentClient.Timeout)
	}

	// Restore original client
	SetDefaultClient(originalClient)
}

func TestNewInsecureClient(t *testing.T) {
	client := NewInsecureClient()
	if client == nil {
		t.Error("NewInsecureClient() returned nil")
		return
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Error("client transport is not *http.Transport")
	}

	if transport.TLSClientConfig == nil {
		t.Error("TLSClientConfig is nil")
	}

	if !transport.TLSClientConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should be true for insecure client")
	}
}

func TestCtxGet(t *testing.T) {
	ctx := context.Background()

	// Test with a simple endpoint
	data, err := CtxGet(ctx, "https://httpbin.org/json")
	if err != nil {
		t.Logf("CtxGet failed (might be expected if offline): %v", err)
	} else if len(data) == 0 {
		t.Error("CtxGet returned empty data")
	}
}

func TestPost(t *testing.T) {
	body := strings.NewReader(`{"test": "data"}`)

	data, err := Post("https://httpbin.org/post", body, WithContentTypeJSON())
	if err != nil {
		t.Logf("Post failed (might be expected if offline): %v", err)
	} else {
		if len(data) == 0 {
			t.Error("Post returned empty data")
		}
		// Verify response contains our test data
		response := string(data)
		if !strings.Contains(response, "test") {
			t.Logf("Response doesn't contain test data: %s", response)
		}
	}
}

func TestCtxPost(t *testing.T) {
	ctx := context.Background()
	body := strings.NewReader(`{"test": "data"}`)

	data, err := CtxPost(ctx, "https://httpbin.org/post", body, WithContentTypeJSON())
	if err != nil {
		t.Logf("CtxPost failed (might be expected if offline): %v", err)
	} else if len(data) == 0 {
		t.Error("CtxPost returned empty data")
	}
}

func TestPatch(t *testing.T) {
	body := strings.NewReader(`{"test": "data"}`)

	data, err := Patch("https://httpbin.org/patch", body, WithContentTypeJSON())
	if err != nil {
		t.Logf("Patch failed (might be expected if offline): %v", err)
	} else if len(data) == 0 {
		t.Error("Patch returned empty data")
	}
}

func TestCtxPatch(t *testing.T) {
	ctx := context.Background()
	body := strings.NewReader(`{"test": "data"}`)

	data, err := CtxPatch(ctx, "https://httpbin.org/patch", body, WithContentTypeJSON())
	if err != nil {
		t.Logf("CtxPatch failed (might be expected if offline): %v", err)
	} else if len(data) == 0 {
		t.Error("CtxPatch returned empty data")
	}
}

func TestDoRequest(t *testing.T) {
	statusCode, content, err := DoRequest("GET", "https://httpbin.org/json", nil)
	if err != nil {
		t.Logf("DoRequest failed (might be expected if offline): %v", err)
	} else {
		if statusCode != 200 {
			t.Errorf("expected status 200, got %d", statusCode)
		}
		if len(content) == 0 {
			t.Error("DoRequest returned empty content")
		}
	}
}

func TestDoRequestWithContext(t *testing.T) {
	ctx := context.Background()

	statusCode, content, err := DoRequestWithContext(ctx, "GET", "https://httpbin.org/json", nil, nil)
	if err != nil {
		t.Logf("DoRequestWithContext failed (might be expected if offline): %v", err)
	} else {
		if statusCode != 200 {
			t.Errorf("expected status 200, got %d", statusCode)
		}
		if len(content) == 0 {
			t.Error("DoRequestWithContext returned empty content")
		}
	}
}

func TestDoRequestWithOptions(t *testing.T) {
	opts := []RequestOption{
		WithContentTypeJSON(),
		WithUserAgent("test-agent"),
	}

	statusCode, content, headers, err := DoRequestWithOptions("GET", "https://httpbin.org/json", opts, nil)
	if err != nil {
		t.Logf("DoRequestWithOptions failed (might be expected if offline): %v", err)
	} else {
		if statusCode != 200 {
			t.Errorf("expected status 200, got %d", statusCode)
		}
		if len(content) == 0 {
			t.Error("DoRequestWithOptions returned empty content")
		}
		if headers == nil {
			t.Error("DoRequestWithOptions returned nil headers")
		}
	}
}

func TestDoRequestWithOptionsError(t *testing.T) {
	// Test with invalid URL
	statusCode, content, headers, err := DoRequestWithOptions("GET", "invalid-url", nil, nil)
	_ = statusCode
	_ = content
	_ = headers
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

func TestDoRequestWithOptionsWithBody(t *testing.T) {
	body := strings.NewReader(`{"test": "data"}`)

	statusCode, content, respHeaders, err := DoRequestWithOptions("POST", "https://httpbin.org/post", []RequestOption{WithContentTypeJSON()}, body)
	_ = respHeaders // unused in test
	if err != nil {
		t.Logf("DoRequestWithOptions with body failed (might be expected if offline): %v", err)
	} else {
		if statusCode != 200 {
			t.Errorf("expected status 200, got %d", statusCode)
		}
		if len(content) == 0 {
			t.Error("DoRequestWithOptions with body returned empty content")
		}
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancel the context

	_, err := CtxGet(ctx, "https://httpbin.org/delay/1")
	if err == nil {
		t.Error("expected error for cancelled context, got nil")
	}
}

func TestTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This should timeout quickly
	_, err := CtxGet(ctx, "https://httpbin.org/delay/2")
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}