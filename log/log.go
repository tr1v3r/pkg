package log

import (
	"context"
	"io"
)

// Flush flush log
func Flush() { defaultLogger.Flush() }

// Close close
func Close() { defaultLogger.Close() }

// SetLevel set output log level
func SetLevel(l Level) { defaultHandler.SetLevel(l) }

// RegisterOutput register log output
func RegisterOutput(out io.Writer) { defaultHandler.RegisterOutput(out) }

// Trace ...
func Trace(format string, args ...interface{}) {
	defaultLogger.Trace(format, args...)
}

// CtxTrace ...
func CtxTrace(_ context.Context, format string, args ...interface{}) {
	defaultLogger.Trace(format, args...)
}

// Debug ...
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

// CtxDebug ...
func CtxDebug(_ context.Context, format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

// Info ...
func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

// CtxInfo ...
func CtxInfo(_ context.Context, format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

// Warn ...
func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

// CtxWarn ...
func CtxWarn(_ context.Context, format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

// Error ...
func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// CtxError ...
func CtxError(_ context.Context, format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// Fatal ...
func Fatal(format string, args ...interface{}) {
	defaultLogger.Fatal(format, args...)
}

// CtxFatal ...
func CtxFatal(_ context.Context, format string, args ...interface{}) {
	defaultLogger.Fatal(format, args...)
}

// Panic ...
func Panic(format string, args ...interface{}) {
	defaultLogger.Panic(format, args...)
}

// CtxPanic ...
func CtxPanic(_ context.Context, format string, args ...interface{}) {
	defaultLogger.Panic(format, args...)
}
