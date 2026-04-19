package log

import (
	"fmt"
	"strings"
	"time"
)

// TextEncoder formats records as human-readable text lines.
type TextEncoder struct {
	color  bool
	layout string
}

// NewTextEncoder creates a text encoder.
// If color is true, ANSI color codes are added (for console output).
func NewTextEncoder(color bool) *TextEncoder {
	return &TextEncoder{color: color, layout: time.RFC3339}
}

// Encode formats a Record as a single text line.
// Format: 2024-01-01T12:00:00Z [INFO] [logID] message key=value key2="val with spaces"
func (e *TextEncoder) Encode(record Record) []byte {
	var buf strings.Builder

	if e.color {
		if c, ok := levelColors[record.Level]; ok {
			buf.WriteString(c)
		}
	}

	buf.WriteString(record.Time.Format(e.layout))
	buf.WriteString(" [")
	buf.WriteString(record.Level.String())
	buf.WriteString("] ")

	if record.LogID != "" {
		buf.WriteString("[")
		buf.WriteString(record.LogID)
		buf.WriteString("] ")
	}

	if record.Caller != "" {
		buf.WriteString(record.Caller)
		buf.WriteString(" ")
	}

	buf.WriteString(record.Message)

	for _, f := range record.Fields {
		buf.WriteString(" ")
		buf.WriteString(f.Key)
		buf.WriteString("=")
		buf.WriteString(formatValue(f.Value))
	}

	if e.color {
		buf.WriteString(reset)
	}

	buf.WriteString("\n")
	return []byte(buf.String())
}

func formatValue(v any) string {
	switch val := v.(type) {
	case string:
		if strings.ContainsAny(val, " \t\n\"") {
			return fmt.Sprintf("%q", val)
		}
		return val
	case error:
		return fmt.Sprintf("%q", val.Error())
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprintf("%v", val)
	}
}

var levelColors = map[Level]string{
	TraceLevel: "\033[38;5;45m",
	DebugLevel: "\033[38;5;39m",
	InfoLevel:  "\033[38;5;33m",
	WarnLevel:  "\033[38;5;148m",
	ErrorLevel: "\033[38;5;161m",
	FatalLevel: "\033[38;5;160m",
}

const reset = "\033[0m"
