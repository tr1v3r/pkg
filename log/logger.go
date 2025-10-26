package log

import (
	"context"
	"io"
	"sync"
)

// contextKey is a type for context keys to avoid collisions
type contextKey string

// LogIDKey is the context key for log ID
const LogIDKey contextKey = "log_id"

// Logger provides methods for all log levels with support for both traditional and structured logging
type Logger interface {
	Trace(format string, args ...any)
	Debug(format string, args ...any)
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
	Fatal(format string, args ...any)
	Panic(format string, args ...any)

	CtxTrace(ctx context.Context, format string, args ...any)
	CtxDebug(ctx context.Context, format string, args ...any)
	CtxInfo(ctx context.Context, format string, args ...any)
	CtxWarn(ctx context.Context, format string, args ...any)
	CtxError(ctx context.Context, format string, args ...any)
	CtxFatal(ctx context.Context, format string, args ...any)
	CtxPanic(ctx context.Context, format string, args ...any)

	With(args ...any) Logger
	WithGroup(name string) Logger

	SetLevel(level Level)
	SetOutput(w io.Writer)
	AddOutputs(writers ...io.Writer)
	Flush()
	Close() error
}

// Handler formats and writes log messages
type Handler interface {
	io.Writer
	Output(level Level, ctx context.Context, format string, args ...any)

	With(args ...any) Handler
	WithGroup(name string) Handler

	SetLevel(level Level)
	SetOutput(w io.Writer)
	AddOutputs(writers ...io.Writer)
	Flush()
	Close() error
}

// Formatter converts log data into string representations
type Formatter interface {
	Format(level Level, ctx context.Context, format string, args ...any) string
}

// baseLogger is the concrete implementation of the Logger interface
type baseLogger struct {
	mu       sync.RWMutex
	handlers []Handler
	level    Level
}

// NewLogger creates a new logger with the specified handlers
// If no handlers are provided, a default console handler is used
func NewLogger(handlers ...Handler) Logger {
	if len(handlers) == 0 {
		handlers = []Handler{NewConsoleHandler(InfoLevel)}
	}

	return &baseLogger{
		handlers: handlers,
		level:    InfoLevel,
	}
}

// SetLevel sets the minimum log level for all handlers
func (l *baseLogger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.level = level
	for _, handler := range l.handlers {
		handler.SetLevel(level)
	}
}

// SetOutput sets the output writer for all handlers
func (l *baseLogger) SetOutput(w io.Writer) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, handler := range l.handlers {
		handler.SetOutput(w)
	}
}

// AddOutputs adds multiple output writers to all handlers
func (l *baseLogger) AddOutputs(writers ...io.Writer) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, handler := range l.handlers {
		handler.AddOutputs(writers...)
	}
}

// Flush flushes all pending log messages
func (l *baseLogger) Flush() {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, handler := range l.handlers {
		handler.Flush()
	}
}

// Close closes all handlers and releases resources
func (l *baseLogger) Close() error {
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
func (l *baseLogger) With(args ...any) Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Create new handlers with structured fields applied
	newHandlers := make([]Handler, len(l.handlers))
	for i, handler := range l.handlers {
		newHandlers[i] = handler.With(args...)
	}

	// Return new logger with updated handlers
	return &baseLogger{
		handlers: newHandlers,
		level:    l.level,
	}
}

// WithGroup creates a new logger that starts a group
func (l *baseLogger) WithGroup(name string) Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Create new handlers with group applied
	newHandlers := make([]Handler, len(l.handlers))
	for i, handler := range l.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}

	// Return new logger with updated handlers
	return &baseLogger{
		handlers: newHandlers,
		level:    l.level,
	}
}

// Level-based logging methods
func (l *baseLogger) Trace(format string, args ...any) {
	l.CtxTrace(context.TODO(), format, args...)
}

func (l *baseLogger) Debug(format string, args ...any) {
	l.CtxDebug(context.TODO(), format, args...)
}

func (l *baseLogger) Info(format string, args ...any) {
	l.CtxInfo(context.TODO(), format, args...)
}

func (l *baseLogger) Warn(format string, args ...any) {
	l.CtxWarn(context.TODO(), format, args...)
}

func (l *baseLogger) Error(format string, args ...any) {
	l.CtxError(context.TODO(), format, args...)
}

func (l *baseLogger) Fatal(format string, args ...any) {
	l.CtxFatal(context.TODO(), format, args...)
}

func (l *baseLogger) Panic(format string, args ...any) {
	l.CtxPanic(context.TODO(), format, args...)
}

// Context-aware logging methods
func (l *baseLogger) CtxTrace(ctx context.Context, format string, args ...any) {
	l.output(TraceLevel, ctx, format, args...)
}

func (l *baseLogger) CtxDebug(ctx context.Context, format string, args ...any) {
	l.output(DebugLevel, ctx, format, args...)
}

func (l *baseLogger) CtxInfo(ctx context.Context, format string, args ...any) {
	l.output(InfoLevel, ctx, format, args...)
}

func (l *baseLogger) CtxWarn(ctx context.Context, format string, args ...any) {
	l.output(WarnLevel, ctx, format, args...)
}

func (l *baseLogger) CtxError(ctx context.Context, format string, args ...any) {
	l.output(ErrorLevel, ctx, format, args...)
}

func (l *baseLogger) CtxFatal(ctx context.Context, format string, args ...any) {
	l.output(FatalLevel, ctx, format, args...)
}

func (l *baseLogger) CtxPanic(ctx context.Context, format string, args ...any) {
	l.output(PanicLevel, ctx, format, args...)
}

// output sends log messages to all handlers
func (l *baseLogger) output(level Level, ctx context.Context, format string, args ...any) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, handler := range l.handlers {
		handler.Output(level, ctx, format, args...)
	}
}

// MultiError represents multiple errors from closing handlers
type MultiError struct {
	Errors []error
}

func (m *MultiError) Error() string {
	if len(m.Errors) == 0 {
		return "no errors"
	}
	return "multiple errors occurred while closing handlers"
}
