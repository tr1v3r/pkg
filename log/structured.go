package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
)

// StructuredLogger provides structured logging with backward compatibility
type StructuredLogger struct {
	mu       sync.RWMutex
	handlers []Handler
	slogger  *slog.Logger
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(handlers ...Handler) *StructuredLogger {
	return &StructuredLogger{
		handlers: handlers,
		slogger:  slog.Default(),
	}
}

// SetStructuredLogger sets the underlying slog logger for structured output
func (l *StructuredLogger) SetStructuredLogger(slogger *slog.Logger) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.slogger = slogger
}

// SetLevel sets the minimum log level for all handlers
func (l *StructuredLogger) SetLevel(level Level) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, handler := range l.handlers {
		handler.SetLevel(level)
	}
}

// SetOutput sets the output writer for all handlers
func (l *StructuredLogger) SetOutput(w io.Writer) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, handler := range l.handlers {
		handler.SetOutput(w)
	}
}

// AddOutput adds an additional output writer to all handlers
func (l *StructuredLogger) AddOutput(w io.Writer) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, handler := range l.handlers {
		handler.AddOutput(w)
	}
}

// Flush flushes all pending log messages
func (l *StructuredLogger) Flush() {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, handler := range l.handlers {
		handler.Flush()
	}
}

// Close closes all handlers and releases resources
func (l *StructuredLogger) Close() error {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var errors []error
	for _, handler := range l.handlers {
		if err := handler.Close(); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return &MultiError{Errors: errors}
	}
	return nil
}

// With creates a new logger with additional structured fields
func (l *StructuredLogger) With(args ...interface{}) Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.slogger == nil {
		return l
	}

	newLogger := &StructuredLogger{
		handlers: l.handlers,
		slogger:  l.slogger.With(args...),
	}
	return newLogger
}

// WithGroup creates a new logger that starts a group
func (l *StructuredLogger) WithGroup(name string) Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.slogger == nil {
		return l
	}

	newLogger := &StructuredLogger{
		handlers: l.handlers,
		slogger:  l.slogger.WithGroup(name),
	}
	return newLogger
}

// Traditional logging methods (format string based)

func (l *StructuredLogger) Trace(format string, args ...interface{}) {
	l.CtxTrace(nil, format, args...)
}

func (l *StructuredLogger) Debug(format string, args ...interface{}) {
	l.CtxDebug(nil, format, args...)
}

func (l *StructuredLogger) Info(format string, args ...interface{}) {
	l.CtxInfo(nil, format, args...)
}

func (l *StructuredLogger) Warn(format string, args ...interface{}) {
	l.CtxWarn(nil, format, args...)
}

func (l *StructuredLogger) Error(format string, args ...interface{}) {
	l.CtxError(nil, format, args...)
}

func (l *StructuredLogger) Fatal(format string, args ...interface{}) {
	l.CtxFatal(nil, format, args...)
}

func (l *StructuredLogger) Panic(format string, args ...interface{}) {
	l.CtxPanic(nil, format, args...)
}

// Context-aware traditional logging methods

func (l *StructuredLogger) CtxTrace(ctx context.Context, format string, args ...interface{}) {
	l.output(TraceLevel, ctx, format, args...)
	l.slog(ctx, slog.LevelDebug-4, format, args...) // Trace is below Debug
}

func (l *StructuredLogger) CtxDebug(ctx context.Context, format string, args ...interface{}) {
	l.output(DebugLevel, ctx, format, args...)
	l.slog(ctx, slog.LevelDebug, format, args...)
}

func (l *StructuredLogger) CtxInfo(ctx context.Context, format string, args ...interface{}) {
	l.output(InfoLevel, ctx, format, args...)
	l.slog(ctx, slog.LevelInfo, format, args...)
}

func (l *StructuredLogger) CtxWarn(ctx context.Context, format string, args ...interface{}) {
	l.output(WarnLevel, ctx, format, args...)
	l.slog(ctx, slog.LevelWarn, format, args...)
}

func (l *StructuredLogger) CtxError(ctx context.Context, format string, args ...interface{}) {
	l.output(ErrorLevel, ctx, format, args...)
	l.slog(ctx, slog.LevelError, format, args...)
}

func (l *StructuredLogger) CtxFatal(ctx context.Context, format string, args ...interface{}) {
	l.output(FatalLevel, ctx, format, args...)
	l.slog(ctx, slog.LevelError+4, format, args...) // Fatal is above Error
}

func (l *StructuredLogger) CtxPanic(ctx context.Context, format string, args ...interface{}) {
	l.output(PanicLevel, ctx, format, args...)
	l.slog(ctx, slog.LevelError+8, format, args...) // Panic is above Fatal
}

// output sends log messages to traditional handlers
func (l *StructuredLogger) output(level Level, ctx context.Context, format string, args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, handler := range l.handlers {
		handler.Output(level, ctx, format, args...)
	}
}

// slog sends structured log using slog
func (l *StructuredLogger) slog(ctx context.Context, level slog.Level, format string, args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.slogger == nil {
		return
	}

	// Convert format string and args to structured logging
	msg := format
	if len(args) > 0 {
		// If we have args, try to format them
		if strings.Contains(format, "%") {
			msg = fmt.Sprintf(format, args...)
			l.slogger.Log(ctx, level, msg)
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
				l.slogger.LogAttrs(ctx, level, msg, attrs...)
			} else {
				// Odd number of args - use as message with extra args
				msg = fmt.Sprintf(format, args...)
				l.slogger.Log(ctx, level, msg)
			}
		}
	} else {
		// No args, just log the message
		l.slogger.Log(ctx, level, msg)
	}
}

