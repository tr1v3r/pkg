package config

import "io"

//	func Init() {
//		if err := c.Load(); err != nil {
//			panic(fmt.Errorf("load config fail: %w", err))
//		}
//	}

func (c *Configure) LoadToFrom(v any, r io.Reader) error { return c.loadTo(v, r) }
