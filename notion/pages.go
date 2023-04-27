package notion

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/riverchu/pkg/fetch"
	"github.com/riverchu/pkg/log"
)

// NewPageManager return a new page manager
func NewPageManager(version, token string) *PageManager {
	return &PageManager{baseInfo: &baseInfo{
		NotionVersion: version,
		BearerToken:   token,
	}}
}

// PageManager ...
type PageManager struct {
	*baseInfo

	ID string
}

// WithID set page id
func (pm PageManager) WithID(id string) *PageManager {
	pm.ID = id
	return &pm
}

// Update update page
// docs: https://developers.notion.com/reference/patch-page
// PATCH https://api.notion.com/v1/pages/{page_id}
func (pm *PageManager) Update(properties ...*Property) error {
	log.Debug("update page %s", pm.ID)

	payload, _ := json.Marshal(map[string]interface{}{"properties": PropertyArray(properties).ForUpdate()})
	log.Debug("update page with payload: %s", string(payload))

	resp, err := fetch.Patch(pm.api(updateOp), bytes.NewBuffer(payload), pm.Headers()...)
	if err != nil {
		return fmt.Errorf("update database %s fail: %w", pm.ID, err)
	}
	log.Debug("update page got response %s", string(resp))

	var obj Object
	if err := json.Unmarshal(resp, &obj); err != nil {
		return fmt.Errorf("unmarshal page %s fail: %w", pm.ID, err)
	}
	// {"object":"error","status":401,"code":"unauthorized","message":"API token is invalid."}
	if obj.Object == "error" {
		return fmt.Errorf("update page fail: [%d / %s] %s", obj.Status, obj.Code, obj.Message)
	}
	return nil
}

// api return database api
func (pm *PageManager) api(typ operateType) string {
	baseAPI := notionAPI() + "/databases"
	switch typ {
	case createOp: // POST https://api.notion.com/v1/pages
		return baseAPI
	case retrieveOp: // GET https://api.notion.com/v1/pages/{page_id}
		return baseAPI + "/" + pm.ID
	case retrievePropOp: // GET https://api.notion.com/v1/pages/{page_id}/properties/{property_id}
		return baseAPI + "/" + pm.ID + "/properties/"
	case updateOp: // PATCH https://api.notion.com/v1/pages/{page_id}
		return baseAPI + "/" + pm.ID
	default:
		return ""
	}
}
