package fetch

import (
	"context"
	"net/http"
)

// RequestOption ...
type RequestOption func(req *http.Request) *http.Request

var (
	// WithHeader ...
	WithHeader = func(key string, values ...string) RequestOption {
		return func(req *http.Request) *http.Request {
			for _, value := range values {
				req.Header.Add(key, value)
			}
			return req
		}
	}

	// WithHeaders ...
	WithHeaders = func(header map[string][]string) RequestOption {
		return func(req *http.Request) *http.Request {
			for key, values := range header {
				for _, value := range values {
					req.Header.Add(key, value)
				}
			}
			return req
		}
	}

	// WithSetHeader set k,v in header
	WithSetHeader = func(key, value string) RequestOption {
		return func(req *http.Request) *http.Request {
			req.Header.Set(key, value)
			return req
		}
	}

	// WithoutHeader ...
	WithoutHeader = func(key string) RequestOption {
		return func(req *http.Request) *http.Request {
			req.Header.Del(key)
			return req
		}
	}

	// WithContentType ...
	WithContentType = func(contentType string) RequestOption {
		return func(req *http.Request) *http.Request {
			req.Header.Set("Content-Type", contentType)
			return req
		}
	}

	// WithContentTypeJSON set content type as json
	WithContentTypeJSON = func() RequestOption {
		return WithContentType("application/json")
	}

	// WithAuthToken set "Authorization" header with token
	WithAuthToken = func(token string) RequestOption {
		return func(req *http.Request) *http.Request {
			req.Header.Set("Authorization", token)
			return req
		}
	}

	// WithContext wrap request with context
	WithContext = func(ctx context.Context) RequestOption {
		return func(req *http.Request) *http.Request {
			return req.WithContext(ctx)
		}
	}
)
