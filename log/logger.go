package log

import (
	"context"
	"io"
	"sync"
)

// Logger provides methods for all log levels with support for both traditional and structured logging
type Logger interface {
	Trace(format string, args ...interface{})
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Fatal(format string, args ...interface{})
	Panic(format string, args ...interface{})

	CtxTrace(ctx context.Context, format string, args ...interface{})
	CtxDebug(ctx context.Context, format string, args ...interface{})
	CtxInfo(ctx context.Context, format string, args ...interface{})
	CtxWarn(ctx context.Context, format string, args ...interface{})
	CtxError(ctx context.Context, format string, args ...interface{})
	CtxFatal(ctx context.Context, format string, args ...interface{})
	CtxPanic(ctx context.Context, format string, args ...interface{})

	With(args ...interface{}) Logger
	WithGroup(name string) Logger

	SetLevel(level Level)
	SetOutput(w io.Writer)
	AddOutput(w io.Writer)
	Flush()
	Close() error
}

// Handler formats and writes log messages
type Handler interface {
	io.Writer
	Output(level Level, ctx context.Context, format string, args ...interface{})
	SetLevel(level Level)
	SetOutput(w io.Writer)
	AddOutput(w io.Writer)
	Flush()
	Close() error
}

// Formatter converts log data into string representations
type Formatter interface {
	Format(level Level, ctx context.Context, format string, args ...interface{}) string
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

// AddOutput adds an additional output writer to all handlers
func (l *baseLogger) AddOutput(w io.Writer) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, handler := range l.handlers {
		handler.AddOutput(w)
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
func (l *baseLogger) With(args ...interface{}) Logger {
	// For base logger, return self (no structured logging support)
	// Structured logging is handled by StructuredLogger wrapper
	return l
}

// WithGroup creates a new logger that starts a group
func (l *baseLogger) WithGroup(name string) Logger {
	// For base logger, return self (no structured logging support)
	return l
}

// Level-based logging methods
func (l *baseLogger) Trace(format string, args ...interface{}) {
	l.CtxTrace(nil, format, args...)
}

func (l *baseLogger) Debug(format string, args ...interface{}) {
	l.CtxDebug(nil, format, args...)
}

func (l *baseLogger) Info(format string, args ...interface{}) {
	l.CtxInfo(nil, format, args...)
}

func (l *baseLogger) Warn(format string, args ...interface{}) {
	l.CtxWarn(nil, format, args...)
}

func (l *baseLogger) Error(format string, args ...interface{}) {
	l.CtxError(nil, format, args...)
}

func (l *baseLogger) Fatal(format string, args ...interface{}) {
	l.CtxFatal(nil, format, args...)
}

func (l *baseLogger) Panic(format string, args ...interface{}) {
	l.CtxPanic(nil, format, args...)
}

// Context-aware logging methods
func (l *baseLogger) CtxTrace(ctx context.Context, format string, args ...interface{}) {
	l.output(TraceLevel, ctx, format, args...)
}

func (l *baseLogger) CtxDebug(ctx context.Context, format string, args ...interface{}) {
	l.output(DebugLevel, ctx, format, args...)
}

func (l *baseLogger) CtxInfo(ctx context.Context, format string, args ...interface{}) {
	l.output(InfoLevel, ctx, format, args...)
}

func (l *baseLogger) CtxWarn(ctx context.Context, format string, args ...interface{}) {
	l.output(WarnLevel, ctx, format, args...)
}

func (l *baseLogger) CtxError(ctx context.Context, format string, args ...interface{}) {
	l.output(ErrorLevel, ctx, format, args...)
}

func (l *baseLogger) CtxFatal(ctx context.Context, format string, args ...interface{}) {
	l.output(FatalLevel, ctx, format, args...)
}

func (l *baseLogger) CtxPanic(ctx context.Context, format string, args ...interface{}) {
	l.output(PanicLevel, ctx, format, args...)
}

// output sends log messages to all handlers
func (l *baseLogger) output(level Level, ctx context.Context, format string, args ...interface{}) {
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