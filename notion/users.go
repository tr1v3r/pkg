package notion

import (
	"context"
	"fmt"

	"github.com/tr1v3r/pkg/log"
)

// UserManager implements UserAPI.
type UserManager struct {
	client *notionClient
}

// NewUserManager creates a UserManager with default settings.
func NewUserManager(version, token string) *UserManager {
	return &UserManager{
		client: newNotionClient(version, token, defaultLimiter()),
	}
}

// Me retrieves the current authenticated bot user.
// GET /v1/users/me
func (um *UserManager) Me(ctx context.Context) (*User, error) {
	log.CtxDebugf(ctx, "retrieve current user")

	var user User
	if err := um.client.get(ctx, "/users/me", &user); err != nil {
		return nil, fmt.Errorf("retrieve current user: %w", err)
	}
	return &user, nil
}

// Retrieve retrieves a user by ID.
// GET /v1/users/{user_id}
func (um *UserManager) Retrieve(ctx context.Context, id string) (*User, error) {
	log.CtxDebugf(ctx, "retrieve user %s", id)

	var user User
	if err := um.client.get(ctx, "/users/"+id, &user); err != nil {
		return nil, fmt.Errorf("retrieve user %s: %w", id, err)
	}
	return &user, nil
}

// List retrieves all users in the workspace.
// GET /v1/users
func (um *UserManager) List(ctx context.Context) ([]User, error) {
	log.CtxDebugf(ctx, "list users")

	return paginateAll[User](ctx, um.client, "GET", "/users", func(cursor string) any {
		return &ListOptions{PageSize: 100, Cursor: cursor}
	})
}
