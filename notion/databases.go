package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/riverchu/pkg/fetch"
	"github.com/riverchu/pkg/log"
)

// NewDatabaseManager return a new database manager
func NewDatabaseManager(version, token string) *DatabaseManager {
	return &DatabaseManager{baseInfo: &baseInfo{
		NotionVersion: version,
		BearerToken:   token,
	}, ctx: context.Background()}
}

// DatabaseManager ...
type DatabaseManager struct {
	*baseInfo

	ID string

	ctx context.Context
}

// WithID set database id
func (dm DatabaseManager) WithID(id string) *DatabaseManager {
	dm.ID = id
	return &dm
}

// WithContext set Context
func (dm DatabaseManager) WithContext(ctx context.Context) *DatabaseManager {
	dm.ctx = ctx
	return &dm
}

// Create create database
// docs: https://developers.notion.com/reference/create-a-database
// POST https://api.notion.com/v1/databases
func (dm *DatabaseManager) Create(parent PageItem, title []TextObject, properties map[string]*Property) error {
	log.Debug("create database")

	payload, _ := json.Marshal(&struct {
		Parent     PageItem             `json:"parent"`
		Icon       *IconItem            `json:"icon,omitempty"`
		Cover      *FileItem            `json:"cover,omitempty"`
		Title      []TextObject         `json:"title"`
		Properties map[string]*Property `json:"properties"`
	}{
		Parent:     parent,
		Title:      title,
		Properties: properties,
	})
	statusCode, resp, _, err := fetch.DoRequestWithOptions("POST", dm.api(createOp),
		append(dm.Headers(), fetch.WithContext(dm.ctx)), bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create database fail: %w", err)
	}
	if statusCode != 200 {
		return fmt.Errorf("create database fail: [%d] %s", statusCode, string(resp))
	}
	return nil
}

// Retrieve retrieve database
// docs: https://developers.notion.com/reference/retrieve-a-database
// GET https://api.notion.com/v1/databases/{database_id}
func (dm *DatabaseManager) Retrieve() (*Object, error) {
	log.Debug("retrieve database %s", dm.ID)

	resp, err := fetch.CtxGet(dm.ctx, dm.api(retrieveOp), dm.Headers()...)
	if err != nil {
		return nil, fmt.Errorf("retrieve database %s fail: %w", dm.ID, err)
	}

	log.Debug("retrieve database got %s", string(resp))

	var obj Object
	if err := json.Unmarshal(resp, &obj); err != nil {
		return nil, fmt.Errorf("unmarshal database %s fail: %w", dm.ID, err)
	}
	// {"object":"error","status":401,"code":"unauthorized","message":"API token is invalid."}
	if obj.Object == "error" {
		return nil, fmt.Errorf("retrieve database fail: [%d / %s] %s", obj.Status, obj.Code, obj.Message)
	}
	return &obj, nil
}

// Query query databases
// docs: https://developers.notion.com/reference/post-database-query
// POST https://api.notion.com/v1/databases/{database_id}/query
func (dm *DatabaseManager) Query(filter *Filter) (results []Object, err error) {
	log.Debug("query database %s", dm.ID)

	var api = dm.api(queryOp)

	resp, err := fetch.CtxPost(dm.ctx, api, bytes.NewReader(filter.Payload()), dm.Headers()...)
	if err != nil {
		return nil, fmt.Errorf("retrieve database %s fail: %w", dm.ID, err)
	}

	var obj = new(Object)
	if err := json.Unmarshal(resp, obj); err != nil {
		return nil, fmt.Errorf("unmarshal database %s fail: %w", dm.ID, err)
	}
	// {"object":"error","status":401,"code":"unauthorized","message":"API token is invalid."}
	if obj.Object == "error" {
		return nil, fmt.Errorf("query database fail: [%d / %s] %s", obj.Status, obj.Code, obj.Message)
	}
	// build a new array for results, or array will be owerwritten because of the same memory address with obj.Results
	results = append(make([]Object, 0, len(obj.Results)), obj.Results...)
	log.Debug("fetch %d items, next cursor: %s", len(obj.Results), obj.NextCursor)

	for obj.HasMore {
		resp, err := fetch.CtxPost(dm.ctx, api, bytes.NewReader((&Filter{StartCursor: obj.NextCursor}).Payload()), dm.Headers()...)
		if err != nil {
			return nil, fmt.Errorf("retrieve database %s fail: %w", dm.ID, err)
		}

		obj = new(Object)
		if err := json.Unmarshal(resp, obj); err != nil {
			return nil, fmt.Errorf("unmarshal database %s fail: %w", dm.ID, err)
		}

		// {"object":"error","status":401,"code":"unauthorized","message":"API token is invalid."}
		if obj.Object == "error" {
			return nil, fmt.Errorf("query database fail: [%d / %s] %s", obj.Status, obj.Code, obj.Message)
		}

		results = append(results, obj.Results...)
		log.Debug("catch %d items, next cursor: %s", len(results), obj.NextCursor)
	}

	log.Debug("query database got %d items", len(results))
	return results, nil
}

// Update update database
// docs: https://developers.notion.com/reference/update-a-database
// PATCH https://api.notion.com/v1/databases/{database_id}
func (dm *DatabaseManager) Update(payload io.Reader) error {
	log.Debug("update database %s", dm.ID)

	resp, err := fetch.CtxPatch(dm.ctx, dm.api(updateOp), payload, dm.Headers()...)
	if err != nil {
		return fmt.Errorf("query api %s fail: %w", dm.ID, err)
	}
	log.Debug("update database got %s", string(resp))

	var obj Object
	if err := json.Unmarshal(resp, &obj); err != nil {
		return fmt.Errorf("unmarshal api response %s fail: %w", dm.ID, err)
	}
	// {"object":"error","status":401,"code":"unauthorized","message":"API token is invalid."}
	if obj.Object == "error" {
		return fmt.Errorf("update fail: [%d / %s] %s", obj.Status, obj.Code, obj.Message)
	}
	return nil
}

// api return database api
func (dm *DatabaseManager) api(typ operateType) string {
	baseAPI := notionAPI() + "/databases"
	switch typ {
	case createOp: // POST https://api.notion.com/v1/databases
		return baseAPI
	case queryOp: // POST https://api.notion.com/v1/databases/{database_id}/query
		return baseAPI + "/" + dm.ID + "/query"
	case retrieveOp: // GET https://api.notion.com/v1/databases/{database_id}
		return baseAPI + "/" + dm.ID
	case updateOp: // PATCH https://api.notion.com/v1/databases/{database_id}
		return baseAPI + "/" + dm.ID
	default:
		return ""
	}
}
