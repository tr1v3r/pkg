package notion

import "errors"

var (
	// ErrRateLimited rate limited
	// https://developers.notion.com/reference/request-limits
	ErrRateLimited = errors.New("notion rate limited")
)
