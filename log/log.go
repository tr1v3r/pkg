package log

import (
	"context"
	"io"
	"log/slog"
)

// Global logger instance
var (
	defaultHandler Handler = NewConsoleHandler(InfoLevel)
	defaultLogger  Logger  = NewStructuredLogger(defaultHandler)
)

// Global convenience functions

// SetLevel sets the minimum log level for the default logger
func SetLevel(level Level) {
	defaultLogger.SetLevel(level)
}

// SetOutput sets the output writer for the default logger
func SetOutput(w io.Writer) {
	defaultLogger.SetOutput(w)
}

// AddOutput adds an additional output writer to the default logger
func AddOutput(w io.Writer) {
	defaultLogger.AddOutput(w)
}

// Flush flushes all pending log messages
func Flush() {
	defaultLogger.Flush()
}

// Close closes the default logger and releases resources
func Close() error {
	return defaultLogger.Close()
}

// WithLogID creates a new context with the specified log ID
// This is a convenience function for users to easily add log IDs to their contexts
func WithLogID(ctx context.Context, logID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, LogIDKey, logID)
}

// Traditional logging functions

// Trace logs a message at TraceLevel
func Trace(format string, args ...any) {
	defaultLogger.Trace(format, args...)
}

// Debug logs a message at DebugLevel
func Debug(format string, args ...any) {
	defaultLogger.Debug(format, args...)
}

// Info logs a message at InfoLevel
func Info(format string, args ...any) {
	defaultLogger.Info(format, args...)
}

// Warn logs a message at WarnLevel
func Warn(format string, args ...any) {
	defaultLogger.Warn(format, args...)
}

// Error logs a message at ErrorLevel
func Error(format string, args ...any) {
	defaultLogger.Error(format, args...)
}

// Fatal logs a message at FatalLevel and exits the program
func Fatal(format string, args ...any) {
	defaultLogger.Fatal(format, args...)
}

// Panic logs a message at PanicLevel and panics
func Panic(format string, args ...any) {
	defaultLogger.Panic(format, args...)
}

// Context-aware logging functions

// CtxTrace logs a message at TraceLevel with context
func CtxTrace(ctx context.Context, format string, args ...any) {
	defaultLogger.CtxTrace(ctx, format, args...)
}

// CtxDebug logs a message at DebugLevel with context
func CtxDebug(ctx context.Context, format string, args ...any) {
	defaultLogger.CtxDebug(ctx, format, args...)
}

// CtxInfo logs a message at InfoLevel with context
func CtxInfo(ctx context.Context, format string, args ...any) {
	defaultLogger.CtxInfo(ctx, format, args...)
}

// CtxWarn logs a message at WarnLevel with context
func CtxWarn(ctx context.Context, format string, args ...any) {
	defaultLogger.CtxWarn(ctx, format, args...)
}

// CtxError logs a message at ErrorLevel with context
func CtxError(ctx context.Context, format string, args ...any) {
	defaultLogger.CtxError(ctx, format, args...)
}

// CtxFatal logs a message at FatalLevel with context and exits the program
func CtxFatal(ctx context.Context, format string, args ...any) {
	defaultLogger.CtxFatal(ctx, format, args...)
}

// CtxPanic logs a message at PanicLevel with context and panics
func CtxPanic(ctx context.Context, format string, args ...any) {
	defaultLogger.CtxPanic(ctx, format, args...)
}

// Structured logging convenience functions

// With creates a new logger with the given structured fields
func With(args ...any) Logger {
	return defaultLogger.With(args...)
}

// WithGroup creates a new logger that starts a group
func WithGroup(name string) Logger {
	return defaultLogger.WithGroup(name)
}

// SetStructuredOutput configures structured logging output
// This sets up the underlying slog logger for JSON or text output
func SetStructuredOutput(out io.Writer, json bool) {
	if sl, ok := defaultLogger.(*StructuredLogger); ok {
		var handler slog.Handler
		if json {
			handler = slog.NewJSONHandler(out, nil)
		} else {
			handler = slog.NewTextHandler(out, nil)
		}
		sl.SetStructuredLogger(slog.New(handler))
	}
}

// Handler management functions

// RegisterHandler adds handlers to the default logger
// This maintains backward compatibility with the original API
func RegisterHandler(handlers ...Handler) {
	if sl, ok := defaultLogger.(*StructuredLogger); ok {
		sl.mu.Lock()
		defer sl.mu.Unlock()
		sl.handlers = append(sl.handlers, handlers...)
	}
}

// ClearHandler removes all handlers from the default logger
// This maintains backward compatibility with the original API
func ClearHandler() {
	if sl, ok := defaultLogger.(*StructuredLogger); ok {
		sl.mu.Lock()
		defer sl.mu.Unlock()
		sl.handlers = make([]Handler, 0, 4)
	}
}
