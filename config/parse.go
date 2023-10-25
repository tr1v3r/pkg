package config

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

// Parser get parser
type Parser = func(any, []byte) error

var (
	// jsonParser parse json config
	JSONParser Parser = func(v any, data []byte) error {
		return json.Unmarshal(data, v)
	}
	// yamlParser parse yaml config
	YAMLParser Parser = func(v any, data []byte) error { // nolint
		return yaml.Unmarshal(data, v)
	}
)
