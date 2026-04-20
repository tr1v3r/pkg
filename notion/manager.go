package notion

import (
	"context"
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
func NewClient(version, token string) *Client {
	limiter := defaultLimiter()
	client := newNotionClient(version, token, limiter)

	return &Client{
		Database: &DatabaseManager{client: client},
		Page:     &PageManager{client: client},
		Block:    &BlockManager{client: client},
		Search:   &SearchManager{client: client},
		User:     &UserManager{client: client},
		Comment:  &CommentManager{client: client},
		client:   client,
	}
}

// Set updates the Notion API version and token.
func (c *Client) Set(version, token string) {
	c.client.version = version
	c.client.token = token
}

// WithContext returns a new Client with the given context.
func (c Client) WithContext(ctx context.Context) *Client {
	c.client = &notionClient{
		version: c.client.version,
		token:   c.client.token,
		limiter: c.client.limiter,
	}
	return &c
}

// WithLimiter returns a new Client with the given rate limiter.
func (c Client) WithLimiter(limiter *rate.Limiter) *Client {
	c.client = &notionClient{
		version: c.client.version,
		token:   c.client.token,
		limiter: limiter,
	}
	return &c
}
