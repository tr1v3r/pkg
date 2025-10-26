package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// structuredLogHandler provides structured logging
// It implements the Handler interface and outputs to slog system
type structuredLogHandler struct {
	mu      sync.RWMutex
	slogger *slog.Logger // Internal slog handler for structured format
	level   Level
	output  io.Writer
}

// NewStructuredLogHandler creates a new structured log handler
func NewStructuredLogHandler(level Level) *structuredLogHandler {
	return &structuredLogHandler{
		slogger: slog.Default(),
		level:   level,
		output:  nil, // Use default output
	}
}

// Output implements Handler.Output - outputs to structured slog system
func (h *structuredLogHandler) Output(level Level, ctx context.Context, format string, args ...any) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Output to slog system
	if h.slogger != nil {
		h.slog(level, ctx, format, args...)
	}

	// Also output to h.output if set
	if h.output != nil {
		// Format the message for text output
		var msg string
		if len(args) > 0 {
			msg = fmt.Sprintf(format, args...)
		} else {
			msg = format
		}

		// Add timestamp and level for text output
		timestamp := time.Now().Format(time.RFC3339)
		levelStr := level.String()
		logMsg := fmt.Sprintf("%s [%s] %s\n", timestamp, levelStr, msg)

		// Write to output
		_, _ = h.output.Write([]byte(logMsg))
	}
}

// SetLevel sets the minimum log level
func (h *structuredLogHandler) SetLevel(level Level) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.level = level
}

// SetOutput sets the output writer
func (h *structuredLogHandler) SetOutput(w io.Writer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.output = w
}

// With returns a new handler with the given structured fields
func (h *structuredLogHandler) With(args ...any) Handler {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Create a new handler with the same configuration
	newHandler := &structuredLogHandler{
		slogger: h.slogger,
		level:   h.level,
		output:  h.output,
	}

	// Apply structured fields to the underlying slog logger
	if h.slogger != nil && len(args) > 0 {
		if len(args)%2 == 0 {
			// Even number of args - treat as key-value pairs
			attrs := make([]any, len(args))
			copy(attrs, args)
			newHandler.slogger = h.slogger.With(attrs...)
		} else {
			// Odd number of args - use as message with extra args
			// For structured logging, we'll still create a new logger
			// but without structured fields
			newHandler.slogger = h.slogger
		}
	}

	return newHandler
}

// WithGroup returns a new handler that starts a group
func (h *structuredLogHandler) WithGroup(name string) Handler {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Create a new handler with the same configuration
	newHandler := &structuredLogHandler{
		slogger: h.slogger,
		level:   h.level,
		output:  h.output,
	}

	// Apply group to the underlying slog logger
	if h.slogger != nil && name != "" {
		newHandler.slogger = h.slogger.WithGroup(name)
	}

	return newHandler
}

// SetStructuredLogger sets the underlying slog logger for structured output
func (h *structuredLogHandler) SetStructuredLogger(slogger *slog.Logger) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.slogger = slogger
}

// slog sends structured log using slog
func (h *structuredLogHandler) slog(level Level, ctx context.Context, format string, args ...any) {
	if h.slogger == nil {
		return
	}

	// Convert log level to slog level
	slogLevel := h.convertLevel(level)

	// Convert format string and args to structured logging
	msg := format
	if len(args) > 0 {
		// If we have args, try to format them
		if strings.Contains(format, "%") {
			msg = fmt.Sprintf(format, args...)
			h.slogger.Log(ctx, slogLevel, msg)
		} else {
			// If no format specifiers, treat args as key-value pairs
			if len(args)%2 == 0 {
				// Even number of args - treat as key-value pairs
				attrs := make([]slog.Attr, 0, len(args)/2)
				for i := 0; i < len(args); i += 2 {
					if key, ok := args[i].(string); ok {
						attrs = append(attrs, slog.Any(key, args[i+1]))
					}
				}
				h.slogger.LogAttrs(ctx, slogLevel, msg, attrs...)
			} else {
				// Odd number of args - use as message with extra args
				msg = fmt.Sprintf(format, args...)
				h.slogger.Log(ctx, slogLevel, msg)
			}
		}
	} else {
		// No args, just log the message
		h.slogger.Log(ctx, slogLevel, msg)
	}
}

// convertLevel converts custom log level to slog level
func (h *structuredLogHandler) convertLevel(level Level) slog.Level {
	switch level {
	case TraceLevel:
		return slog.LevelDebug - 4 // Trace is below Debug
	case DebugLevel:
		return slog.LevelDebug
	case InfoLevel:
		return slog.LevelInfo
	case WarnLevel:
		return slog.LevelWarn
	case ErrorLevel:
		return slog.LevelError
	case FatalLevel:
		return slog.LevelError + 4 // Fatal is above Error
	case PanicLevel:
		return slog.LevelError + 8 // Panic is above Fatal
	default:
		return slog.LevelInfo
	}
}

// AddOutputs adds multiple output writers
func (h *structuredLogHandler) AddOutputs(writers ...io.Writer) {
	// For structured logging, we don't support multiple outputs
	// This method exists to satisfy the Handler interface
}

// Flush flushes any pending log messages
func (h *structuredLogHandler) Flush() {
	// For structured logging, flushing is handled by the underlying slog system
	// This method exists to satisfy the Handler interface
}

// Close closes the handler and releases resources
func (h *structuredLogHandler) Close() error {
	// For structured logging, no resources need to be closed
	// This method exists to satisfy the Handler interface
	return nil
}

// Write implements io.Writer interface
func (h *structuredLogHandler) Write(p []byte) (n int, err error) {
	// For structured logging, we don't support direct writing
	// This method exists to satisfy the io.Writer interface
	return len(p), nil
}
