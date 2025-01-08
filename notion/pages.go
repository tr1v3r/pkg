package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/time/rate"

	"github.com/tr1v3r/pkg/fetch"
	"github.com/tr1v3r/pkg/log"
)

// NewPageManager return a new page manager
func NewPageManager(version, token string) *PageManager {
	return &PageManager{baseInfo: &baseInfo{
		NotionVersion: version,
		BearerToken:   token,
	}, ctx: context.Background(), limiter: rate.NewLimiter(rateLimit, 4*rateLimit)}
}

// PageManager ...
type PageManager struct {
	*baseInfo

	ctx     context.Context
	id      string
	limiter *rate.Limiter
}

// WithContext set Context
func (pm PageManager) WithContext(ctx context.Context) *PageManager {
	pm.ctx = ctx
	return &pm
}

// WithID set page id
func (pm PageManager) WithID(id string) *PageManager {
	pm.id = id
	return &pm
}

// WithLimiter with limiiter
func (pm PageManager) WithLimiter(limiter *rate.Limiter) *PageManager {
	pm.limiter = limiter
	return &pm
}

// ID get page id
func (pm *PageManager) ID() string {
	return pm.id
}

// Create create page
// docs: https://developers.notion.com/reference/post-page
// POST https://api.notion.com/v1/pages
func (pm *PageManager) Create(parent PageItem, properties ...*Property) error {
	log.CtxDebug(pm.ctx, "create page from parent: %+v", parent)

	payload, _ := json.Marshal(&struct {
		Parent     PageItem `json:"parent"`
		Properties any      `json:"properties"`
	}{
		Parent:     parent,
		Properties: PropertyArray(properties).ForUpdate(),
	})
	_ = pm.limiter.Wait(pm.ctx)
	statusCode, resp, _, err := fetch.DoRequestWithOptions("POST", pm.api(createOp),
		append(pm.Headers(), fetch.WithContext(pm.ctx)), bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("request api fail: %w", err)
	}
	if statusCode != 200 {
		return fmt.Errorf("response error: [%d] %s", statusCode, string(resp))
	}
	return nil
}

// Update update page
// docs: https://developers.notion.com/reference/patch-page
// PATCH https://api.notion.com/v1/pages/{page_id}
func (pm *PageManager) Update(properties ...*Property) error {
	log.CtxDebug(pm.ctx, "update page %s", pm.id)

	payload, _ := json.Marshal(map[string]any{"properties": PropertyArray(properties).ForUpdate()})
	log.CtxDebug(pm.ctx, "update page with payload: %s", string(payload))

	_ = pm.limiter.Wait(pm.ctx)
	resp, err := fetch.CtxPatch(pm.ctx, pm.api(updateOp), bytes.NewReader(payload), pm.Headers()...)
	if err != nil {
		return fmt.Errorf("request api fail: %w", err)
	}
	log.CtxDebug(pm.ctx, "update page got response %s", string(resp))

	var obj Object
	if err := json.Unmarshal(resp, &obj); err != nil {
		return fmt.Errorf("unmarshal response fail: %w", err)
	}
	// {"object":"error","status":401,"code":"unauthorized","message":"API token is invalid."}
	if obj.Object == "error" {
		return fmt.Errorf("response error: [%d / %s] %s", obj.Status, obj.Code, obj.Message)
	}
	return nil
}

// Trash trash a page
// https://developers.notion.com/reference/archive-a-page
func (pm *PageManager) Trash() error {
	log.CtxDebug(pm.ctx, "trash page %s", pm.id)

	// archived or in_trash
	payload, _ := json.Marshal(map[string]any{"in_trash": true})

	_ = pm.limiter.Wait(pm.ctx)
	resp, err := fetch.CtxPatch(pm.ctx, pm.api(updateOp), bytes.NewReader(payload), pm.Headers()...)
	if err != nil {
		return fmt.Errorf("request api fail: %w", err)
	}
	log.CtxDebug(pm.ctx, "trash page got response %s", string(resp))

	var obj Object
	if err := json.Unmarshal(resp, &obj); err != nil {
		return fmt.Errorf("unmarshal response fail: %w", err)
	}
	// {"object":"error","status":401,"code":"unauthorized","message":"API token is invalid."}
	if obj.Object == "error" {
		return fmt.Errorf("response error: [%d / %s] %s", obj.Status, obj.Code, obj.Message)
	}
	return nil
}

// api return page api
func (pm *PageManager) api(typ operateType) string {
	baseAPI := notionAPI() + "/pages"
	switch typ {
	case createOp: // POST https://api.notion.com/v1/pages
		return baseAPI
	case retrieveOp: // GET https://api.notion.com/v1/pages/{page_id}
		return baseAPI + "/" + pm.id
	case retrievePropOp: // GET https://api.notion.com/v1/pages/{page_id}/properties/{property_id}
		return baseAPI + "/" + pm.id + "/properties/"
	case updateOp: // PATCH https://api.notion.com/v1/pages/{page_id}
		return baseAPI + "/" + pm.id
	default:
		return ""
	}
}
