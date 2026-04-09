package fetch

import (
	"context"
	"net/http"
	"time"
)

// RequestOption ...
type RequestOption func(req *http.Request) *http.Request

type contextKey string

const (
	maxBodySizeKey   contextKey = "maxBodySize"
	timeoutCancelKey contextKey = "timeoutCancel"
	middlewareKey    contextKey = "middlewares"
)

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

	// WithUserAgent set User-Agent header
	WithUserAgent = func(userAgent string) RequestOption {
		return func(req *http.Request) *http.Request {
			req.Header.Set("User-Agent", userAgent)
			return req
		}
	}

	// WithBasicAuth set basic authentication
	WithBasicAuth = func(username, password string) RequestOption {
		return func(req *http.Request) *http.Request {
			req.SetBasicAuth(username, password)
			return req
		}
	}

	// WithTimeout set request timeout.
	// The underlying context cancel is automatically called after the request completes.
	WithTimeout = func(timeout time.Duration) RequestOption {
		return func(req *http.Request) *http.Request {
			ctx, cancel := context.WithTimeout(req.Context(), timeout)
			ctx = context.WithValue(ctx, timeoutCancelKey, cancel)
			return req.WithContext(ctx)
		}
	}

	// WithQueryParams set URL query parameters
	WithQueryParams = func(params map[string]string) RequestOption {
		return func(req *http.Request) *http.Request {
			q := req.URL.Query()
			for key, value := range params {
				q.Add(key, value)
			}
			req.URL.RawQuery = q.Encode()
			return req
		}
	}

	// WithMaxResponseBodySize sets the maximum response body size in bytes.
	// Defaults to 100MB if not specified. Set to -1 for unlimited.
	WithMaxResponseBodySize = func(maxSize int64) RequestOption {
		return func(req *http.Request) *http.Request {
			ctx := context.WithValue(req.Context(), maxBodySizeKey, maxSize)
			return req.WithContext(ctx)
		}
	}

	// WithMiddleware applies middleware(s) to the request pipeline.
	// Middlewares are applied in order, wrapping the actual HTTP execution.
	WithMiddleware = func(middlewares ...Middleware) RequestOption {
		return func(req *http.Request) *http.Request {
			existing, _ := req.Context().Value(middlewareKey).([]Middleware)
			all := append(existing, middlewares...)
			ctx := context.WithValue(req.Context(), middlewareKey, all)
			return req.WithContext(ctx)
		}
	}
)
