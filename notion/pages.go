package notion

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tr1v3r/pkg/log"
)

// PageManager implements PageAPI.
type PageManager struct {
	client *notionClient
}

// NewPageManager creates a PageManager with default settings.
func NewPageManager(version, token string) *PageManager {
	return &PageManager{
		client: newNotionClient(version, token, defaultLimiter()),
	}
}

// Retrieve retrieves a page by ID.
// GET /v1/pages/{page_id}
func (pm *PageManager) Retrieve(ctx context.Context, id string) (*Page, error) {
	log.CtxDebugf(ctx, "retrieve page %s", id)

	var page Page
	if err := pm.client.get(ctx, "/pages/"+id, &page); err != nil {
		return nil, fmt.Errorf("retrieve page %s: %w", id, err)
	}
	return &page, nil
}

// RetrieveProperty retrieves a page property item, handling pagination automatically.
// GET /v1/pages/{page_id}/properties/{property_id}
func (pm *PageManager) RetrieveProperty(ctx context.Context, pageID, propertyID string) (*Property, error) {
	log.CtxDebugf(ctx, "retrieve page %s property %s", pageID, propertyID)

	path := "/pages/" + pageID + "/properties/" + propertyID
	var prop Property
	if err := pm.client.get(ctx, path, &prop); err != nil {
		return nil, fmt.Errorf("retrieve property %s/%s: %w", pageID, propertyID, err)
	}
	return &prop, nil
}

// Create creates a new page.
// POST /v1/pages
func (pm *PageManager) Create(ctx context.Context, parent ParentRef, properties ...*Property) (*Page, error) {
	log.CtxDebugf(ctx, "create page from parent: %+v", parent)

	body := &struct {
		Parent     ParentRef         `json:"parent"`
		Properties json.RawMessage   `json:"properties"`
	}{
		Parent:     parent,
		Properties: PropertyArray(properties).ForUpdate(),
	}

	var page Page
	if err := pm.client.post(ctx, "/pages", body, &page); err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}
	return &page, nil
}

// Update updates page properties.
// PATCH /v1/pages/{page_id}
func (pm *PageManager) Update(ctx context.Context, id string, properties ...*Property) (*Page, error) {
	log.CtxDebugf(ctx, "update page %s", id)

	body := map[string]any{"properties": PropertyArray(properties).ForUpdate()}

	var page Page
	if err := pm.client.patch(ctx, "/pages/"+id, body, &page); err != nil {
		return nil, fmt.Errorf("update page %s: %w", id, err)
	}
	return &page, nil
}

// Trash moves a page to trash.
// PATCH /v1/pages/{page_id} with in_trash: true
func (pm *PageManager) Trash(ctx context.Context, id string) (*Page, error) {
	log.CtxDebugf(ctx, "trash page %s", id)

	body := map[string]any{"in_trash": true}

	var page Page
	if err := pm.client.patch(ctx, "/pages/"+id, body, &page); err != nil {
		return nil, fmt.Errorf("trash page %s: %w", id, err)
	}
	return &page, nil
}
