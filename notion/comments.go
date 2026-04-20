package notion

import (
	"context"
	"fmt"

	"github.com/tr1v3r/pkg/log"
)

// CommentManager implements CommentAPI.
type CommentManager struct {
	client *notionClient
}

// NewCommentManager creates a CommentManager with default settings.
func NewCommentManager(version, token string) *CommentManager {
	return &CommentManager{
		client: newNotionClient(version, token, defaultLimiter()),
	}
}

// List retrieves comments for a block or page.
// GET /v1/comments?block_id={block_id}
func (cm *CommentManager) List(ctx context.Context, blockID string) ([]Comment, error) {
	log.CtxDebugf(ctx, "list comments for block %s", blockID)

	return paginateAll[Comment](ctx, cm.client, "GET", "/comments?block_id="+blockID, func(cursor string) any {
		return &ListOptions{PageSize: 100, Cursor: cursor}
	})
}

// Create creates a comment on a page or block.
// POST /v1/comments
func (cm *CommentManager) Create(ctx context.Context, parent ParentRef, text []TextObject) (*Comment, error) {
	log.CtxDebugf(ctx, "create comment on %+v", parent)

	body := map[string]any{
		"parent":    parent,
		"rich_text": text,
	}

	var comment Comment
	if err := cm.client.post(ctx, "/comments", body, &comment); err != nil {
		return nil, fmt.Errorf("create comment: %w", err)
	}
	return &comment, nil
}
