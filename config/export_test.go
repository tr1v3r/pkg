package config_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/tr1v3r/pkg/config"
	"github.com/tr1v3r/pkg/guard"
)

// export functions

func GetName() string { return c.Name }

func Cancel() <-chan struct{}  { return guard.Cancel() }
func Cancelled() bool          { return guard.Cancelled() }
func Context() context.Context { return guard.Context() }

// configure
var c = &MyConfig{
	ctx: guard.Context(),
	C:   config.NewConfigure(),
}

type MyConfig struct {
	ctx context.Context   `json:"-"`
	C   *config.Configure `json:"-"`

	Name string `json:"name"`
}

func (c *MyConfig) Load(paths ...string) error { return c.C.LoadTo(c, paths...) }
func (c *MyConfig) LoadFrom(r io.Reader) error { return c.C.LoadToFrom(c, r) }

func TestGetName(t *testing.T) {
	err := c.LoadFrom(bytes.NewReader([]byte(`{"name":"jing"}`)))
	if err != nil {
		t.Errorf("load from io.Reader fail: %s", err)
	}

	name := GetName()
	t.Logf("got name from config: %s", name)
}
