package notion

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/riverchu/pkg/log"
	"github.com/sirupsen/logrus"
)

var (
	version = os.Getenv("NOTION_VERSION")
	token   = os.Getenv("NOTION_TOKEN")
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.DebugLevel)
}

func TestRetrieve_Database(t *testing.T) {
	mgr := NewManager(version, token)
	mgr.DatabaseManager.ID = os.Getenv("NOTION_DATABASE_ID")
	obj, err := mgr.DatabaseManager.Retrieve()
	if err != nil {
		t.Errorf("query fail: %s", err)
		return
	}

	data, _ := json.Marshal(obj)
	t.Logf("retrieve database success: %s", string(data))
}

func TestQuery_Database_all(t *testing.T) {
	mgr := NewManager(version, token)
	mgr.DatabaseManager.ID = os.Getenv("NOTION_DATABASE_ID")

	// query all
	results, err := mgr.DatabaseManager.Query(nil)
	if err != nil {
		t.Errorf("query fail: %s", err)
		return
	}
	t.Logf("got %d results", len(results))
	for _, result := range results {
		t.Logf("got %s\n", result.Properties["Name"])
	}

	// data, _ := json.Marshal(obj)
	// t.Logf("query database all items success: %s", string(data))
}
