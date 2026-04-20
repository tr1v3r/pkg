package notion

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/tr1v3r/pkg/log"
)

var (
	version    = os.Getenv("NOTION_VERSION")
	token      = os.Getenv("NOTION_TOKEN")
	databaseID = os.Getenv("NOTION_DATABASE_ID")
)

func init() {
	log.Setup(log.ConsoleTo(os.Stdout, log.WithLevel(log.DebugLevel)))
}

func TestRetrieve_Database(t *testing.T) {
	mgr := NewManager(version, token)
	db, err := mgr.Database.Retrieve(context.Background(), databaseID)
	if err != nil {
		t.Fatalf("retrieve fail: %s", err)
	}

	data, _ := json.Marshal(db)
	t.Logf("retrieve database success: %s", string(data))
}

func TestQuery_Database_all(t *testing.T) {
	mgr := NewManager(version, token)

	results, err := mgr.Database.Query(context.Background(), databaseID, &Condition{
		Sorts: []PropSortCondition{{Property: "总市值", Direction: "descending"}},
	})
	if err != nil {
		t.Fatalf("query fail: %s", err)
	}
	t.Logf("got %d results", len(results))
	for _, result := range results {
		t.Logf("got %v\n", result.Properties["Name"])
	}
}

func TestCreate_Page(t *testing.T) {
	mgr := NewManager(version, token)

	data, _ := json.Marshal([]TextObject{{
		Text:        TextItem{Content: "000001"},
		Annotations: &Annotation{Bold: true, Color: "default"},
	}})
	page, err := mgr.Page.Create(context.Background(),
		PageItem{DatabaseID: databaseID},
		&Property{Name: "Code", Type: RichTextProp, RichText: data},
	)
	if err != nil {
		t.Fatalf("create page fail: %s", err)
	}
	t.Logf("created page: %s", page.ID)
}
