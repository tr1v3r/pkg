package log

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// Logger dispatches Records to multiple Sinks.
type Logger struct {
	sinks  []*Sink
	fields []Field
	mu     sync.RWMutex
}

// New creates a Logger that writes to the given sinks.
func New(sinks ...*Sink) *Logger {
	if len(sinks) == 0 {
		sinks = []*Sink{Console()}
	}
	return &Logger{sinks: sinks}
}

// With returns a child Logger that carries preset fields.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		sinks:  l.sinks,
		fields: append(l.fields, toFields(args...)...),
	}
}

// --- Structured logging (no context) ---

// Trace logs at the trace level with structured key-value pairs.
func (l *Logger) Trace(msg string, args ...any) { l.log(TraceLevel, msg, args) }

// Debug logs at the debug level with structured key-value pairs.
func (l *Logger) Debug(msg string, args ...any) { l.log(DebugLevel, msg, args) }

// Info logs at the info level with structured key-value pairs.
func (l *Logger) Info(msg string, args ...any) { l.log(InfoLevel, msg, args) }

// Warn logs at the warn level with structured key-value pairs.
func (l *Logger) Warn(msg string, args ...any) { l.log(WarnLevel, msg, args) }

// Error implements the error interface.
func (l *Logger) Error(msg string, args ...any) { l.log(ErrorLevel, msg, args) }

// Fatal logs at the fatal level and exits with status 1 (after flushing sinks).
func (l *Logger) Fatal(msg string, args ...any) { l.log(FatalLevel, msg, args); l.exit() }

// --- Structured logging (with context, extracts logID) ---

// Trace logs with the logID extracted from ctx.
func (l *Logger) CtxTrace(ctx context.Context, msg string, args ...any) {
	l.logCtx(ctx, TraceLevel, msg, args)
}

// Debug logs with the logID extracted from ctx.
func (l *Logger) CtxDebug(ctx context.Context, msg string, args ...any) {
	l.logCtx(ctx, DebugLevel, msg, args)
}

// Info logs with the logID extracted from ctx.
func (l *Logger) CtxInfo(ctx context.Context, msg string, args ...any) {
	l.logCtx(ctx, InfoLevel, msg, args)
}

// Warn logs with the logID extracted from ctx.
func (l *Logger) CtxWarn(ctx context.Context, msg string, args ...any) {
	l.logCtx(ctx, WarnLevel, msg, args)
}

// Error logs with the logID extracted from ctx.
func (l *Logger) CtxError(ctx context.Context, msg string, args ...any) {
	l.logCtx(ctx, ErrorLevel, msg, args)
}

// Fatal logs with the logID extracted from ctx.
func (l *Logger) CtxFatal(ctx context.Context, msg string, args ...any) {
	l.logCtx(ctx, FatalLevel, msg, args)
	l.exit()
}

// --- Printf-style ---

// Trace logs a printf-style message.
func (l *Logger) Tracef(format string, args ...any) { l.logf(TraceLevel, format, args) }

// Debug logs a printf-style message.
func (l *Logger) Debugf(format string, args ...any) { l.logf(DebugLevel, format, args) }

// Info logs a printf-style message.
func (l *Logger) Infof(format string, args ...any) { l.logf(InfoLevel, format, args) }

// Warn logs a printf-style message.
func (l *Logger) Warnf(format string, args ...any) { l.logf(WarnLevel, format, args) }

// Error logs a printf-style message.
func (l *Logger) Errorf(format string, args ...any) { l.logf(ErrorLevel, format, args) }

// Fatal logs a printf-style message.
func (l *Logger) Fatalf(format string, args ...any) { l.logf(FatalLevel, format, args); l.exit() }

// --- Printf-style with context (extracts logID) ---

// Tracef logs with the logID extracted from ctx.
func (l *Logger) CtxTracef(ctx context.Context, format string, args ...any) {
	l.logCtxf(ctx, TraceLevel, format, args)
}

// Debugf logs with the logID extracted from ctx.
func (l *Logger) CtxDebugf(ctx context.Context, format string, args ...any) {
	l.logCtxf(ctx, DebugLevel, format, args)
}

// Infof logs with the logID extracted from ctx.
func (l *Logger) CtxInfof(ctx context.Context, format string, args ...any) {
	l.logCtxf(ctx, InfoLevel, format, args)
}

// Warnf logs with the logID extracted from ctx.
func (l *Logger) CtxWarnf(ctx context.Context, format string, args ...any) {
	l.logCtxf(ctx, WarnLevel, format, args)
}

// Errorf logs with the logID extracted from ctx.
func (l *Logger) CtxErrorf(ctx context.Context, format string, args ...any) {
	l.logCtxf(ctx, ErrorLevel, format, args)
}

// Fatalf logs with the logID extracted from ctx.
func (l *Logger) CtxFatalf(ctx context.Context, format string, args ...any) {
	l.logCtxf(ctx, FatalLevel, format, args)
	l.exit()
}

// exit flushes all sinks before terminating: os.Exit skips defers and async
// sinks would otherwise lose the fatal record that is still buffered.
func (l *Logger) exit() {
	l.Sync()
	os.Exit(1)
}

// --- Internal ---

func (l *Logger) log(level Level, msg string, args []any) {
	l.dispatch(Record{
		Time:    time.Now(),
		Level:   level,
		Message: msg,
		Fields:  mergeFields(l.fields, toFields(args...)),
	})
}

func (l *Logger) logCtx(ctx context.Context, level Level, msg string, args []any) {
	l.dispatch(Record{
		Time:    time.Now(),
		Level:   level,
		Message: msg,
		Fields:  mergeFields(l.fields, toFields(args...)),
		LogID:   extractLogID(ctx),
	})
}

func (l *Logger) logCtxf(ctx context.Context, level Level, format string, args []any) {
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}
	l.dispatch(Record{
		Time:    time.Now(),
		Level:   level,
		Message: msg,
		Fields:  l.fields,
		LogID:   extractLogID(ctx),
	})
}

func (l *Logger) logf(level Level, format string, args []any) {
	l.dispatch(Record{
		Time:    time.Now(),
		Level:   level,
		Message: fmt.Sprintf(format, args...),
		Fields:  l.fields,
	})
}

func (l *Logger) dispatch(record Record) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, s := range l.sinks {
		s.Log(record)
	}
}

func mergeFields(base, extra []Field) []Field {
	if len(base) == 0 {
		return extra
	}
	if len(extra) == 0 {
		return base
	}
	merged := make([]Field, 0, len(base)+len(extra))
	merged = append(merged, base...)
	merged = append(merged, extra...)
	return merged
}

// Sync flushes all sinks.
func (l *Logger) Sync() {
	for _, s := range l.sinks {
		s.Sync()
	}
}

// Close closes all sinks.
func (l *Logger) Close() error {
	var firstErr error
	for _, s := range l.sinks {
		if err := s.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// SetLevel sets the minimum level for all sinks.
func (l *Logger) SetLevel(level Level) {
	for _, s := range l.sinks {
		s.SetLevel(level)
	}
}

// ============================================================
// Global logger — package-level convenience functions
// ============================================================

var (
	globalMu     sync.RWMutex
	globalLogger = New(Console())
)

// Setup replaces the global logger with new sinks.
func Setup(sinks ...*Sink) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalLogger = New(sinks...)
}

// SetLevel sets the minimum level for all global sinks.
func SetLevel(level Level) { globalLogger.SetLevel(level) }

// With returns a child of the global logger with preset fields.
func With(args ...any) *Logger { return globalLogger.With(args...) }

// --- Structured (no context) ---

// Trace logs at the trace level via the global logger.
func Trace(msg string, args ...any) { globalLogger.Trace(msg, args...) }

// Debug logs at the debug level via the global logger.
func Debug(msg string, args ...any) { globalLogger.Debug(msg, args...) }

// Info logs at the info level via the global logger.
func Info(msg string, args ...any) { globalLogger.Info(msg, args...) }

// Warn logs at the warn level via the global logger.
func Warn(msg string, args ...any) { globalLogger.Warn(msg, args...) }

// Error implements the error interface.
func Error(msg string, args ...any) { globalLogger.Error(msg, args...) }

// Fatal logs at the fatal level via the global logger and exits with status 1.
func Fatal(msg string, args ...any) { globalLogger.Fatal(msg, args...) }

// --- Structured with context (extracts logID from ctx) ---

// Trace logs with the logID extracted from ctx.
func CtxTrace(ctx context.Context, msg string, args ...any) { globalLogger.CtxTrace(ctx, msg, args...) }

// Debug logs with the logID extracted from ctx.
func CtxDebug(ctx context.Context, msg string, args ...any) { globalLogger.CtxDebug(ctx, msg, args...) }

// Info logs with the logID extracted from ctx.
func CtxInfo(ctx context.Context, msg string, args ...any) { globalLogger.CtxInfo(ctx, msg, args...) }

// Warn logs with the logID extracted from ctx.
func CtxWarn(ctx context.Context, msg string, args ...any) { globalLogger.CtxWarn(ctx, msg, args...) }

// Error logs with the logID extracted from ctx.
func CtxError(ctx context.Context, msg string, args ...any) { globalLogger.CtxError(ctx, msg, args...) }

// Fatal logs with the logID extracted from ctx.
func CtxFatal(ctx context.Context, msg string, args ...any) { globalLogger.CtxFatal(ctx, msg, args...) }

// --- Printf-style ---

// Trace logs a printf-style message.
func Tracef(format string, args ...any) { globalLogger.Tracef(format, args...) }

// Debug logs a printf-style message.
func Debugf(format string, args ...any) { globalLogger.Debugf(format, args...) }

// Info logs a printf-style message.
func Infof(format string, args ...any) { globalLogger.Infof(format, args...) }

// Warn logs a printf-style message.
func Warnf(format string, args ...any) { globalLogger.Warnf(format, args...) }

// Error logs a printf-style message.
func Errorf(format string, args ...any) { globalLogger.Errorf(format, args...) }

// Fatal logs a printf-style message.
func Fatalf(format string, args ...any) { globalLogger.Fatalf(format, args...) }

// --- Printf-style with context (extracts logID from ctx) ---

// Tracef logs with the logID extracted from ctx.
func CtxTracef(ctx context.Context, format string, args ...any) {
	globalLogger.CtxTracef(ctx, format, args...)
}

// Debugf logs with the logID extracted from ctx.
func CtxDebugf(ctx context.Context, format string, args ...any) {
	globalLogger.CtxDebugf(ctx, format, args...)
}

// Infof logs with the logID extracted from ctx.
func CtxInfof(ctx context.Context, format string, args ...any) {
	globalLogger.CtxInfof(ctx, format, args...)
}

// Warnf logs with the logID extracted from ctx.
func CtxWarnf(ctx context.Context, format string, args ...any) {
	globalLogger.CtxWarnf(ctx, format, args...)
}

// Errorf logs with the logID extracted from ctx.
func CtxErrorf(ctx context.Context, format string, args ...any) {
	globalLogger.CtxErrorf(ctx, format, args...)
}

// Fatalf logs with the logID extracted from ctx.
func CtxFatalf(ctx context.Context, format string, args ...any) {
	globalLogger.CtxFatalf(ctx, format, args...)
}

// Sync flushes all global sinks.
func Sync() { globalLogger.Sync() }

// Close closes all global sinks.
func Close() error { return globalLogger.Close() }
