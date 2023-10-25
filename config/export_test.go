package config_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/tr1v3r/pkg/config"
)

// export functions

func GetName() string { return c.Name }

// configure

var c = &MyConfig{
	C: config.NewConfigure(),
}

type MyConfig struct {
	C *config.Configure `json:"-"`

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
