package config

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/tr1v3r/pkg/fetch"
)

// config struct

// NewConfigure create new configure
// default parser is JSONParser
func NewConfigure() *Configure {
	return &Configure{
		Parser: JSONParser,
	}
}

// Configure ...
type Configure struct{ Parser }

// WithParser set parser
func (c *Configure) WithParser(p Parser) { c.Parser = p }

// Load load config and parse to c
// check env, input param
func (c *Configure) LoadTo(v any, paths ...string) error {
	var path string

	// get path
	if len(paths) > 0 {
		path = paths[0]
	} else {
		path = os.Getenv(EnvConfFile)
	}
	path = strings.TrimSpace(path)

	var r io.Reader

	// fetch file/URL
	if strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "http://") {
		u, err := url.Parse(path)
		if err != nil {
			return fmt.Errorf("invalid URL: %w", err)
		}

		data, err := fetch.Get(u.String())
		if err != nil {
			return fmt.Errorf("fetch URL fail: %w", err)
		}

		r = bytes.NewReader(data)
	} else {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read file fail: %w", err)
		}

		r = bytes.NewReader(data)
	}

	return c.LoadFromTo(r, v)
}

// LoadFromTo load from r and parse to v
func (c *Configure) LoadFromTo(r io.Reader, v any) error {
	buf, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read config fali: %w", err)
	}
	return c.Parser(v, buf)
}
