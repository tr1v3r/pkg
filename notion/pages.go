package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

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

// Retrieve retrieve a page
// docs: https://developers.notion.com/reference/retrieve-a-page
func (pm *PageManager) Retrieve() (*Object, error) {
	log.CtxDebug(pm.ctx, "retrieve page %s", pm.id)

	_ = pm.limiter.Wait(pm.ctx)
	resp, err := fetch.CtxGet(pm.ctx, pm.api(retrieveOp), pm.Headers()...)
	if err != nil {
		return nil, fmt.Errorf("request api fail: %w", err)
	}
	log.CtxDebug(pm.ctx, "retrieve page got response %s", string(resp))

	var obj Object
	if err := json.Unmarshal(resp, &obj); err != nil {
		return nil, fmt.Errorf("unmarshal response fail: %w", err)
	}
	return &obj, nil
}

// RetrieveProp retrieve page property item
// docs: https://developers.notion.com/reference/retrieve-a-page-property
func (pm *PageManager) RetrieveProp(propID string) (*Object, error) {
	log.CtxDebug(pm.ctx, "retrieve page %s property %s", pm.id, propID)

	const pageSize = 30
	param := url.Values{}
	param.Set("page_size", strconv.Itoa(pageSize))

	var obj, results = new(Object), make([]Object, 0, 2*pageSize)
	for obj.HasMore = true; obj.HasMore; {
		if obj.NextCursor != "" {
			param.Set("start_cursor", obj.NextCursor)
		}

		_ = pm.limiter.Wait(pm.ctx)
		resp, err := fetch.CtxGet(pm.ctx, pm.api(retrievePropOp)+propID+"?"+param.Encode(), pm.Headers()...)
		if err != nil {
			return nil, fmt.Errorf("request api fail: %w", err)
		}
		log.CtxDebug(pm.ctx, "retrieve page property got response %s", string(resp))

		if err := json.Unmarshal(resp, obj); err != nil {
			return nil, fmt.Errorf("unmarshal response fail: %w", err)
		}

		results = append(results, obj.Results...)
	}
	obj.Results = results
	return obj, nil
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

	if statusCode == 429 {
		return ErrRateLimited
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
		if obj.Status == 429 {
			return ErrRateLimited
		}
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
		if obj.Status == 429 {
			return ErrRateLimited
		}
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
