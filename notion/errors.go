package notion

import (
	"errors"
	"fmt"
)

var (
	// ErrRateLimited rate limited
	// https://developers.notion.com/reference/request-limits
	ErrRateLimited = errors.New("notion rate limited")
)

// APIError represents an error response from the Notion API.
type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("notion api error: [%d/%s] %s", e.Status, e.Code, e.Message)
}
