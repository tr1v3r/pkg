package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/tr1v3r/pkg/log"
)

// DatabaseManager implements DatabaseAPI.
type DatabaseManager struct {
	client *notionClient
}

// NewDatabaseManager creates a DatabaseManager with default settings.
func NewDatabaseManager(version, token string) *DatabaseManager {
	return &DatabaseManager{
		client: newNotionClient(version, token, defaultLimiter()),
	}
}

// Create creates a database.
// POST /v1/databases
func (dm *DatabaseManager) Create(ctx context.Context, parent ParentRef, title []TextObject, properties map[string]*Property) (*Database, error) {
	log.CtxDebugf(ctx, "create database")

	body := &struct {
		Parent     ParentRef           `json:"parent"`
		Title      []TextObject        `json:"title"`
		Properties map[string]*Property `json:"properties"`
	}{
		Parent:     parent,
		Title:      title,
		Properties: properties,
	}

	var db Database
	if err := dm.client.post(ctx, "/databases", body, &db); err != nil {
		return nil, fmt.Errorf("create database: %w", err)
	}
	return &db, nil
}

// Retrieve retrieves a database by ID.
// GET /v1/databases/{database_id}
func (dm *DatabaseManager) Retrieve(ctx context.Context, id string) (*Database, error) {
	log.CtxDebugf(ctx, "retrieve database %s", id)

	var db Database
	if err := dm.client.get(ctx, "/databases/"+id, &db); err != nil {
		return nil, fmt.Errorf("retrieve database %s: %w", id, err)
	}
	return &db, nil
}

// Query queries a database with filters and sorts, returning all pages.
// POST /v1/databases/{database_id}/query
func (dm *DatabaseManager) Query(ctx context.Context, id string, cond *Condition) ([]Page, error) {
	log.CtxDebugf(ctx, "query database %s", id)

	if cond == nil {
		cond = &Condition{}
	}
	if cond.PageSize <= 0 {
		cond.PageSize = 100
	}

	path := "/databases/" + id + "/query"
	if qp := cond.QueryParams(); qp != "" {
		path += "?" + qp
	}

	return paginateAll[Page](ctx, dm.client, "POST", path, func(cursor string) any {
		c := *cond
		c.StartCursor = cursor
		return c
	})
}

// Update updates a database.
// PATCH /v1/databases/{database_id}
func (dm *DatabaseManager) Update(ctx context.Context, id string, payload io.Reader) (*Database, error) {
	log.CtxDebugf(ctx, "update database %s", id)

	var raw json.RawMessage
	if payload != nil {
		buf, _ := io.ReadAll(payload)
		raw = json.RawMessage(buf)
	}

	var db Database
	if err := dm.client.patch(ctx, "/databases/"+id, raw, &db); err != nil {
		return nil, fmt.Errorf("update database %s: %w", id, err)
	}
	return &db, nil
}
