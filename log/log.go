package log

import (
	"context"
	"io"
)

// Flush flush log
func Flush() { defaultLogger.Flush() }

// Close close
func Close() { defaultLogger.Close() }

// RegisterHandler register handlers
func RegisterHandler(handlers ...Handler) { defaultLogger.RegisterHandler(handlers...) }

// ClearHandler clear all handlers
func ClearHandler() { defaultLogger.ClearHandler() }

// SetLevel set output log level
func SetLevel(l Level) { defaultHandler.SetLevel(l) }

// RegisterOutput register log output
func RegisterOutput(out io.Writer) { defaultHandler.RegisterOutput(out) }

// Trace ...
func Trace(format string, args ...interface{}) {
	defaultLogger.Trace(format, args...)
}

// CtxTrace ...
func CtxTrace(ctx context.Context, format string, args ...interface{}) {
	defaultLogger.CtxTrace(ctx, format, args...)
}

// Debug ...
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

// CtxDebug ...
func CtxDebug(ctx context.Context, format string, args ...interface{}) {
	defaultLogger.CtxDebug(ctx, format, args...)
}

// Info ...
func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

// CtxInfo ...
func CtxInfo(ctx context.Context, format string, args ...interface{}) {
	defaultLogger.CtxInfo(ctx, format, args...)
}

// Warn ...
func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

// CtxWarn ...
func CtxWarn(ctx context.Context, format string, args ...interface{}) {
	defaultLogger.CtxWarn(ctx, format, args...)
}

// Error ...
func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// CtxError ...
func CtxError(ctx context.Context, format string, args ...interface{}) {
	defaultLogger.CtxError(ctx, format, args...)
}

// Fatal ...
func Fatal(format string, args ...interface{}) {
	defaultLogger.Fatal(format, args...)
}

// CtxFatal ...
func CtxFatal(ctx context.Context, format string, args ...interface{}) {
	defaultLogger.CtxFatal(ctx, format, args...)
}

// Panic ...
func Panic(format string, args ...interface{}) {
	defaultLogger.Panic(format, args...)
}

// CtxPanic ...
func CtxPanic(ctx context.Context, format string, args ...interface{}) {
	defaultLogger.CtxPanic(ctx, format, args...)
}
