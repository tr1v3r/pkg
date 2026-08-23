package config_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tr1v3r/pkg/config"
)

type testConf struct {
	Name string `json:"name" yaml:"name"`
	Age  int    `json:"age" yaml:"age"`
}

func TestNewConfigure_DefaultJSON(t *testing.T) {
	c := config.NewConfigure()
	if c == nil {
		t.Fatal("NewConfigure returned nil")
	}

	var v testConf
	if err := c.LoadFromTo(strings.NewReader(`{"name":"a","age":1}`), &v); err != nil {
		t.Fatalf("load default json fail: %s", err)
	}
	if v.Name != "a" || v.Age != 1 {
		t.Errorf("loaded = %+v, want {a 1}", v)
	}
}

func TestConfigure_WithParserYAML(t *testing.T) {
	c := config.NewConfigure()
	c.WithParser(config.YAMLParser)

	var v testConf
	if err := c.LoadFromTo(strings.NewReader("{name: b, age: 2}"), &v); err != nil {
		t.Fatalf("load yaml fail: %s", err)
	}
	if v.Name != "b" || v.Age != 2 {
		t.Errorf("loaded = %+v, want {b 2}", v)
	}
}

func TestConfigure_LoadToFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "conf.json")
	if err := os.WriteFile(path, []byte(`{"name":"file","age":3}`), 0o600); err != nil {
		t.Fatalf("write temp config fail: %s", err)
	}

	var v testConf
	if err := config.NewConfigure().LoadTo(&v, path); err != nil {
		t.Fatalf("load from file fail: %s", err)
	}
	if v.Name != "file" || v.Age != 3 {
		t.Errorf("loaded = %+v, want {file 3}", v)
	}
}

func TestConfigure_LoadToFromEnvPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "env.json")
	if err := os.WriteFile(path, []byte(`{"name":"env","age":4}`), 0o600); err != nil {
		t.Fatalf("write temp config fail: %s", err)
	}

	t.Setenv("CONF_FILE", path)

	var v testConf
	if err := config.NewConfigure().LoadTo(&v); err != nil {
		t.Fatalf("load from env path fail: %s", err)
	}
	if v.Name != "env" || v.Age != 4 {
		t.Errorf("loaded = %+v, want {env 4}", v)
	}
}

func TestConfigure_LoadToFromURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"name":"url","age":5}`)
	}))
	defer srv.Close()

	var v testConf
	if err := config.NewConfigure().LoadTo(&v, srv.URL); err != nil {
		t.Fatalf("load from url fail: %s", err)
	}
	if v.Name != "url" || v.Age != 5 {
		t.Errorf("loaded = %+v, want {url 5}", v)
	}
}

func TestConfigure_LoadToErrors(t *testing.T) {
	var v testConf
	c := config.NewConfigure()

	if err := c.LoadTo(&v, filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Error("missing file should return error")
	}

	// unparsable URL: invalid port
	if err := c.LoadTo(&v, "http://[::1]:notaport"); err == nil {
		t.Error("invalid URL should return error")
	}

	// unreachable host
	if err := c.LoadTo(&v, "http://127.0.0.1:1"); err == nil {
		t.Error("unreachable URL should return error")
	}

	// parser failure: invalid json
	if err := c.LoadFromTo(strings.NewReader("{bad json"), &v); err == nil {
		t.Error("invalid json should return error")
	}
}

type errReader struct{}

func (errReader) Read(_ []byte) (int, error) { return 0, os.ErrClosed }

func TestConfigure_LoadFromToReaderError(t *testing.T) {
	var v testConf
	if err := config.NewConfigure().LoadFromTo(errReader{}, &v); err == nil {
		t.Error("failing reader should return error")
	}
}
