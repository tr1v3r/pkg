package log

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Formatter log formatter
type Formatter interface {
	Format(l Level, ctx context.Context, format string) string
}

// NewStreamFormatter create new stream formatter
func NewStreamFormatter(color bool) Formatter { return &StreamFormatter{color: color} }

// StreamFormattera stream formatter
type StreamFormatter struct {
	color bool
}

// Format format log
func (f *StreamFormatter) Format(l Level, ctx context.Context, format string) string {
	var buf strings.Builder

	if f.color {
		buf.WriteString("\033[38;5;")
		buf.WriteString(fmt.Sprint(l.Color()))
		buf.WriteString("m")
	}

	buf.WriteString(time.Now().Format(time.RFC3339))
	buf.WriteByte(' ')

	buf.WriteByte('[')
	buf.WriteString(strings.ToUpper(l.String()))
	buf.WriteByte(']')
	buf.WriteByte(' ')

	if logID := f.getLogID(ctx); logID != "" {
		buf.WriteString(logID)
		buf.WriteByte(' ')
	}

	buf.WriteString(format)

	if f.color {
		buf.WriteString("\033[0m")
	}

	buf.WriteByte('\n')

	return buf.String()
}

func (f *StreamFormatter) getLogID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	return ctx.Value("log_id").(string)
}
