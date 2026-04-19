package log

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Level represents log severity.
type Level int8

const (
	TraceLevel Level = iota - 1
	DebugLevel
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

func (l Level) String() string {
	switch l {
	case TraceLevel:
		return "TRACE"
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Field is a structured key-value pair attached to a log record.
type Field struct {
	Key   string
	Value any
}

// Record is the immutable data unit that flows through the logging pipeline.
type Record struct {
	Time    time.Time
	Level   Level
	Message string
	Fields  []Field
	LogID   string
	Caller  string
}

// context key
type ctxKey string

const logIDKey ctxKey = "log_id"

// WithLogID stores a logID in the context for later extraction by CtxInfo etc.
func WithLogID(ctx context.Context, logID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, logIDKey, logID)
}

// NewLogID generates a UUID v4 string suitable for request tracing.
func NewLogID() string {
	return uuid.New().String()
}

func extractLogID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(logIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// toFields converts alternating key-value pairs into []Field.
func toFields(args ...any) []Field {
	if len(args) == 0 {
		return nil
	}
	fields := make([]Field, 0, len(args)/2)
	for i := 0; i+1 < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}
		fields = append(fields, Field{Key: key, Value: args[i+1]})
	}
	return fields
}
