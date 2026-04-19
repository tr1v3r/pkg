package log

import (
	"encoding/json"
	"time"
)

// JSONEncoder formats records as JSON lines.
type JSONEncoder struct {
	layout string
}

// NewJSONEncoder creates a JSON encoder.
func NewJSONEncoder() *JSONEncoder {
	return &JSONEncoder{layout: time.RFC3339}
}

// Encode formats a Record as a JSON line.
func (e *JSONEncoder) Encode(record Record) []byte {
	obj := jsonRecord{
		Time:  record.Time.Format(e.layout),
		Level: record.Level.String(),
		Msg:   record.Message,
	}
	if record.LogID != "" {
		obj.LogID = record.LogID
	}
	if record.Caller != "" {
		obj.Caller = record.Caller
	}
	if len(record.Fields) > 0 {
		obj.Fields = make(map[string]any, len(record.Fields))
		for _, f := range record.Fields {
			obj.Fields[f.Key] = f.Value
		}
	}
	data, _ := json.Marshal(obj)
	return append(data, '\n')
}

type jsonRecord struct {
	Time   string         `json:"time"`
	Level  string         `json:"level"`
	Msg    string         `json:"msg"`
	LogID  string         `json:"log_id,omitempty"`
	Caller string         `json:"caller,omitempty"`
	Fields map[string]any `json:"fields,omitempty"`
}
