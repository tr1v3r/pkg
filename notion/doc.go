// Package notion provides a Go client for the Notion API (https://developers.notion.com).
// It offers type-safe, interface-driven access to Databases, Pages, Blocks, Users,
// Comments, and Search endpoints with built-in rate limiting and automatic pagination.
//
// # Getting Started
//
// Create a [Client] with your Notion API version and integration token:
//
//	mgr := notion.NewClient("2022-06-28", "your-notion-token")
//
// The [Client] exposes six domain-specific API interfaces:
//
//   - [DatabaseAPI] — create, retrieve, query, and update databases
//   - [PageAPI] — create, retrieve, update, trash pages, and retrieve page properties
//   - [BlockAPI] — retrieve, update, delete blocks; list and append child blocks
//   - [UserAPI] — retrieve the current bot user, list workspace users, retrieve by ID
//   - [SearchAPI] — search pages and databases in the workspace
//   - [CommentAPI] — list and create comments on pages and blocks
//
// All methods accept [context.Context] as the first argument for cancellation and timeouts.
//
// # Database Operations
//
// Query a database with filters and sorts (pagination is handled automatically):
//
//	pages, err := mgr.Database.Query(ctx, "database-id", &notion.Condition{
//	    Filter: &notion.FilterCondition{
//	        FilterSingleCondition: notion.FilterSingleCondition{
//	            Property: "Status",
//	            Status:   &notion.StatusFilter{Equals: "Done"},
//	        },
//	    },
//	    Sorts: []notion.PropSortCondition{
//	        {Property: "Created", Direction: "descending"},
//	    },
//	})
//
// Retrieve or create databases:
//
//	db, err := mgr.Database.Retrieve(ctx, "database-id")
//	db, err := mgr.Database.Create(ctx, notion.ParentRef{PageID: "parent-page-id"}, title, properties)
//
// # Page Operations
//
// Create a page with properties:
//
//	page, err := mgr.Page.Create(ctx,
//	    notion.ParentRef{DatabaseID: "db-id"},
//	    &notion.Property{Name: "Name", Type: notion.TitleProp, Title: titleJSON},
//	    &notion.Property{Name: "Done", Type: notion.CheckboxProp, Checkbox: ptrBool(true)},
//	)
//
// Retrieve, update, or trash pages:
//
//	page, err := mgr.Page.Retrieve(ctx, "page-id")
//	page, err := mgr.Page.Update(ctx, "page-id", props...)
//	page, err := mgr.Page.Trash(ctx, "page-id")
//
// # Block Operations
//
// Retrieve blocks and their children (pagination is handled automatically):
//
//	block, err := mgr.Block.Retrieve(ctx, "block-id")
//	children, err := mgr.Block.Children(ctx, "block-id", nil)
//
// Append child blocks:
//
//	block, err := mgr.Block.AppendChildren(ctx, "parent-id", []notion.Block{
//	    {Type: "paragraph", Paragraph: &notion.RichTextBlock{
//	        RichText: []notion.TextObject{{Text: notion.TextItem{Content: "Hello world"}}},
//	    }},
//	})
//
// # User and Comment Operations
//
//	me, _ := mgr.User.Me(ctx)
//	user, _ := mgr.User.Retrieve(ctx, "user-id")
//	users, _ := mgr.User.List(ctx)
//
//	comments, _ := mgr.Comment.List(ctx, "page-id")
//	comment, _ := mgr.Comment.Create(ctx, notion.ParentRef{PageID: "page-id"}, []notion.TextObject{
//	    {Text: notion.TextItem{Content: "Nice work!"}},
//	})
//
// # Search
//
//	results, err := mgr.Search.Search(ctx, "keyword", &notion.SearchFilter{
//	    Property: "object",
//	    Value:    "page",
//	})
//
// # Rate Limiting
//
// The client enforces the Notion API rate limit of 3 requests per second by default.
// Customize with [WithRateLimiter]:
//
//	mgr := notion.NewClient("2022-06-28", "token",
//	    notion.WithRateLimiter(rate.NewLimiter(10, 20)),
//	)
//
// # Mock Testing
//
// All Client fields are interface types ([DatabaseAPI], [PageAPI], [BlockAPI],
// [UserAPI], [SearchAPI], [CommentAPI]), enabling straightforward mock testing:
//
//	type mockDB struct{ notion.DatabaseAPI }
//
//	func (m *mockDB) Query(_ context.Context, _ string, _ *notion.Condition) ([]notion.Page, error) {
//	    return []notion.Page{{ID: "mock-page"}}, nil
//	}
//
//	testMgr := &notion.Client{Database: &mockDB{}}
//
// # Error Handling
//
// API errors are returned as [*APIError] with Status, Code, and Message fields.
// Rate limit responses (HTTP 429) return the [ErrRateLimited] sentinel error.
//
//	var apiErr *notion.APIError
//	if errors.As(err, &apiErr) {
//	    fmt.Println(apiErr.Status, apiErr.Code, apiErr.Message)
//	}
package notion
