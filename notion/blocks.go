package notion

import (
	"context"
	"fmt"

	"github.com/tr1v3r/pkg/log"
)

// BlockManager implements BlockAPI.
type BlockManager struct {
	client *notionClient
}

// NewBlockManager creates a BlockManager with default settings.
func NewBlockManager(version, token string) *BlockManager {
	return &BlockManager{
		client: newNotionClient(version, token, defaultLimiter()),
	}
}

// Retrieve retrieves a block by ID.
// GET /v1/blocks/{block_id}
func (bm *BlockManager) Retrieve(ctx context.Context, id string) (*Block, error) {
	log.CtxDebugf(ctx, "retrieve block %s", id)

	var block Block
	if err := bm.client.get(ctx, "/blocks/"+id, &block); err != nil {
		return nil, fmt.Errorf("retrieve block %s: %w", id, err)
	}
	return &block, nil
}

// Update updates a block's content.
// PATCH /v1/blocks/{block_id}
func (bm *BlockManager) Update(ctx context.Context, id string, block *Block) (*Block, error) {
	log.CtxDebugf(ctx, "update block %s", id)

	var result Block
	if err := bm.client.patch(ctx, "/blocks/"+id, block, &result); err != nil {
		return nil, fmt.Errorf("update block %s: %w", id, err)
	}
	return &result, nil
}

// Delete moves a block to trash.
// DELETE /v1/blocks/{block_id}
func (bm *BlockManager) Delete(ctx context.Context, id string) (*Block, error) {
	log.CtxDebugf(ctx, "delete block %s", id)

	var block Block
	if err := bm.client.delete(ctx, "/blocks/"+id, &block); err != nil {
		return nil, fmt.Errorf("delete block %s: %w", id, err)
	}
	return &block, nil
}

// Children retrieves child blocks, handling pagination automatically.
// GET /v1/blocks/{block_id}/children
func (bm *BlockManager) Children(ctx context.Context, blockID string, opts *ListOptions) ([]Block, error) {
	log.CtxDebugf(ctx, "retrieve block %s children", blockID)

	return paginateAll[Block](ctx, bm.client, "GET", "/blocks/"+blockID+"/children", func(cursor string) any {
		if opts == nil {
			return &ListOptions{Cursor: cursor}
		}
		o := *opts
		o.Cursor = cursor
		return &o
	})
}

// AppendChildren appends child blocks to a parent block.
// POST /v1/blocks/{block_id}/children
func (bm *BlockManager) AppendChildren(ctx context.Context, blockID string, children []Block) (*Block, error) {
	log.CtxDebugf(ctx, "append %d children to block %s", len(children), blockID)

	body := map[string]any{"children": children}
	var block Block
	if err := bm.client.post(ctx, "/blocks/"+blockID+"/children", body, &block); err != nil {
		return nil, fmt.Errorf("append children to block %s: %w", blockID, err)
	}
	return &block, nil
}
