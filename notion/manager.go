package notion

import (
	"fmt"

	"golang.org/x/time/rate"
)

const (
	notionAPIHostScheme = "https"
	notionAPIHost       = "api.notion.com"
	apiBasePath         = "v1"
)

func notionAPI() string {
	return fmt.Sprintf("%s://%s/%s", notionAPIHostScheme, notionAPIHost, apiBasePath)
}

func defaultLimiter() *rate.Limiter {
	return rate.NewLimiter(rateLimit, 4*rateLimit)
}

// ClientOption configures a [Client].
type ClientOption func(*Client)

// WithRateLimiter sets a custom rate limiter. Defaults to 3 requests/second with burst of 12.
func WithRateLimiter(limiter *rate.Limiter) ClientOption {
	return func(c *Client) { c.client.limiter = limiter }
}

// Client is the top-level facade for the Notion API.
type Client struct {
	Database DatabaseAPI
	Page     PageAPI
	Block    BlockAPI
	Search   SearchAPI
	User     UserAPI
	Comment  CommentAPI

	client *notionClient
}

// NewClient creates a Client with the given Notion API version and token.
func NewClient(version, token string, opts ...ClientOption) *Client {
	nc := newNotionClient(version, token, defaultLimiter())

	c := &Client{
		Database: &DatabaseManager{client: nc},
		Page:     &PageManager{client: nc},
		Block:    &BlockManager{client: nc},
		Search:   &SearchManager{client: nc},
		User:     &UserManager{client: nc},
		Comment:  &CommentManager{client: nc},
		client:   nc,
	}

	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Set updates the Notion API version and token.
func (c *Client) Set(version, token string) {
	c.client.version = version
	c.client.token = token
}
