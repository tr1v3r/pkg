package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/time/rate"

	"github.com/tr1v3r/pkg/fetch"
	"github.com/tr1v3r/pkg/log"
)

// NewDatabaseManager return a new database manager
func NewDatabaseManager(version, token string) *DatabaseManager {
	return &DatabaseManager{baseInfo: &baseInfo{
		NotionVersion: version,
		BearerToken:   token,
	}, ctx: context.Background(), limiter: rate.NewLimiter(rateLimit, 4*rateLimit)}
}

// DatabaseManager ...
type DatabaseManager struct {
	*baseInfo

	ctx     context.Context
	id      string
	limiter *rate.Limiter
}

// WithContext set Context
func (dm DatabaseManager) WithContext(ctx context.Context) *DatabaseManager {
	dm.ctx = ctx
	return &dm
}

// WithID set database id
func (dm DatabaseManager) WithID(id string) *DatabaseManager {
	dm.id = id
	return &dm
}

// WithLimiter with limiiter
func (dm DatabaseManager) WithLimiter(limiter *rate.Limiter) *DatabaseManager {
	dm.limiter = limiter
	return &dm
}

// ID get database id
func (dm *DatabaseManager) ID() string {
	return dm.id
}

// Create create database
// docs: https://developers.notion.com/reference/create-a-database
// POST https://api.notion.com/v1/databases
func (dm *DatabaseManager) Create(parent PageItem, title []TextObject, properties map[string]*Property) error {
	log.CtxDebug(dm.ctx, "create database")

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

	_ = dm.limiter.Wait(dm.ctx)
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
	log.CtxDebug(dm.ctx, "retrieve database %s", dm.id)

	_ = dm.limiter.Wait(dm.ctx)
	resp, err := fetch.CtxGet(dm.ctx, dm.api(retrieveOp), dm.Headers()...)
	if err != nil {
		return nil, fmt.Errorf("retrieve database %s fail: %w", dm.id, err)
	}

	log.CtxDebug(dm.ctx, "retrieve database got %s", string(resp))

	var obj Object
	if err := json.Unmarshal(resp, &obj); err != nil {
		return nil, fmt.Errorf("unmarshal database %s fail: %w", dm.id, err)
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
func (dm *DatabaseManager) Query(cond *Condition) (objects []Object, err error) {
	for ch, errCh := dm.asyncQuery(cond); ; {
		select {
		case obj, ok := <-ch:
			if !ok {
				return objects, nil
			}
			objects = append(objects, obj)
		case err := <-errCh:
			return nil, err
		}
	}
}

// AsynQuery ...
func (dm *DatabaseManager) AsynQuery(cond *Condition) <-chan Object {
	ch, _ := dm.asyncQuery(cond)
	return ch
}

// asyncQuery query databases in async mode
// docs: https://developers.notion.com/reference/post-database-query
// POST https://api.notion.com/v1/databases/{database_id}/query
func (dm *DatabaseManager) asyncQuery(cond *Condition) (<-chan Object, <-chan error) {
	const defaultPageSize = 100

	log.CtxDebug(dm.ctx, "query database %s", dm.id)

	if cond == nil {
		cond = new(Condition)
	}
	if cond.PageSize <= 0 {
		cond.PageSize = defaultPageSize
	}

	ch := make(chan Object, cond.PageSize)
	errCh := make(chan error, 1)

	output := func(objs []Object) {
		for _, obj := range objs {
			ch <- obj
		}
	}

	go func() {
		// defer close(errCh)
		defer close(ch)
		var count int
		var api = dm.api(queryOp) + "?" + cond.QueryParams()

		var obj = new(Object)
		for obj.HasMore = true; obj.HasMore; {
			cond.StartCursor = obj.NextCursor
			_ = dm.limiter.Wait(dm.ctx)
			resp, err := fetch.CtxPost(dm.ctx, api, bytes.NewReader(cond.Payload()), dm.Headers()...)
			if err != nil {
				// log.CtxError(dm.ctx, "retrieve database %s fail: %s", dm.id, err)
				errCh <- fmt.Errorf("retrieve database %s fail: %w", dm.id, err)
				return
			}

			obj = new(Object)
			if err := json.Unmarshal(resp, obj); err != nil {
				// log.CtxError(dm.ctx, "unmarshal database %s fail: %s\n%s", dm.id, err, resp)
				errCh <- fmt.Errorf("unmarshal database %s fail: %w", dm.id, err)
				return
			}

			// demo: {"object":"error","status":401,"code":"unauthorized","message":"API token is invalid."}
			if obj.Object == "error" {
				// log.CtxError(dm.ctx, "retrieve database %s fail: %s", dm.id, err)
				errCh <- fmt.Errorf("query database fail: [%d / %s] %s", obj.Status, obj.Code, obj.Message)
				return
			}

			output(obj.Results)
			count += len(obj.Results)
			log.CtxDebug(dm.ctx, "total fetched %d items, next cursor: %s", count, obj.NextCursor)
		}

		log.CtxDebug(dm.ctx, "query database got %d items", count)
	}()
	return ch, errCh
}

// Update update database
// docs: https://developers.notion.com/reference/update-a-database
// PATCH https://api.notion.com/v1/databases/{database_id}
func (dm *DatabaseManager) Update(payload io.Reader) error {
	log.CtxDebug(dm.ctx, "update database %s", dm.id)

	_ = dm.limiter.Wait(dm.ctx)
	resp, err := fetch.CtxPatch(dm.ctx, dm.api(updateOp), payload, dm.Headers()...)
	if err != nil {
		return fmt.Errorf("query api %s fail: %w", dm.id, err)
	}
	log.CtxDebug(dm.ctx, "update database got %s", string(resp))

	var obj Object
	if err := json.Unmarshal(resp, &obj); err != nil {
		return fmt.Errorf("unmarshal api response %s fail: %w", dm.id, err)
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
		return baseAPI + "/" + dm.id + "/query"
	case retrieveOp: // GET https://api.notion.com/v1/databases/{database_id}
		return baseAPI + "/" + dm.id
	case updateOp: // PATCH https://api.notion.com/v1/databases/{database_id}
		return baseAPI + "/" + dm.id
	default:
		return ""
	}
}
