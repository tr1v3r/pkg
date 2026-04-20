package notion

import (
	"context"
	"io"
)

// DatabaseAPI defines database operations against the Notion API.
type DatabaseAPI interface {
	Create(ctx context.Context, parent ParentRef, title []TextObject, properties map[string]*Property) (*Database, error)
	Retrieve(ctx context.Context, id string) (*Database, error)
	Query(ctx context.Context, id string, cond *Condition) ([]Page, error)
	Update(ctx context.Context, id string, payload io.Reader) (*Database, error)
}

// PageAPI defines page operations against the Notion API.
type PageAPI interface {
	Create(ctx context.Context, parent ParentRef, properties ...*Property) (*Page, error)
	Retrieve(ctx context.Context, id string) (*Page, error)
	Update(ctx context.Context, id string, properties ...*Property) (*Page, error)
	Trash(ctx context.Context, id string) (*Page, error)
	RetrieveProperty(ctx context.Context, pageID, propertyID string) (*Property, error)
}

// BlockAPI defines block operations against the Notion API.
type BlockAPI interface {
	Retrieve(ctx context.Context, id string) (*Block, error)
	Update(ctx context.Context, id string, block *Block) (*Block, error)
	Delete(ctx context.Context, id string) (*Block, error)
	Children(ctx context.Context, blockID string, opts *ListOptions) ([]Block, error)
	AppendChildren(ctx context.Context, blockID string, children []Block) (*Block, error)
}

// UserAPI defines user operations against the Notion API.
type UserAPI interface {
	Me(ctx context.Context) (*User, error)
	Retrieve(ctx context.Context, id string) (*User, error)
	List(ctx context.Context) ([]User, error)
}

// SearchAPI defines search operations against the Notion API.
type SearchAPI interface {
	Search(ctx context.Context, query string, filter *SearchFilter) (*ListResponse[SearchResult], error)
}

// CommentAPI defines comment operations against the Notion API.
type CommentAPI interface {
	List(ctx context.Context, blockID string) ([]Comment, error)
	Create(ctx context.Context, parent ParentRef, text []TextObject) (*Comment, error)
}

// ListOptions controls pagination for list endpoints.
type ListOptions struct {
	PageSize int    `json:"page_size,omitempty"`
	Cursor   string `json:"start_cursor,omitempty"`
}
