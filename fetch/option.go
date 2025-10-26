package fetch

import (
	"context"
	"net/http"
	"time"
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

	// WithTimeout set request timeout
	WithTimeout = func(timeout time.Duration) RequestOption {
		return func(req *http.Request) *http.Request {
			// This sets the timeout on the context
			ctx, cancel := context.WithTimeout(req.Context(), timeout)
			_ = cancel // The cancel function should be called by the caller
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
)
