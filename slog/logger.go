//nolint:goprintffuncname // All methods must match log.Logger interface naming
package slog

import (
	"context"
	"io"
	stdslog "log/slog"

	"github.com/tr1v3r/pkg/log"
)

// ExtendedLogger extends the existing log.Logger interface to support slog compatibility
type ExtendedLogger interface {
	log.Logger

	// slog.Logger compatibility methods
	Log(ctx context.Context, level stdslog.Level, msg string, args ...any)
	LogAttrs(ctx context.Context, level stdslog.Level, msg string, attrs ...stdslog.Attr)
	DebugCtx(ctx context.Context, msg string, args ...any)
	InfoCtx(ctx context.Context, msg string, args ...any)
	WarnCtx(ctx context.Context, msg string, args ...any)
	ErrorCtx(ctx context.Context, msg string, args ...any)
}

// LoggerWrapper wraps a log.Logger to provide slog compatibility
type LoggerWrapper struct {
	logger log.Logger
}

// NewLoggerWrapper creates a new wrapper that provides slog compatibility for existing log.Logger
func NewLoggerWrapper(logger log.Logger) *LoggerWrapper {
	return &LoggerWrapper{
		logger: logger,
	}
}

// slog.Logger compatibility methods

// Log implements slog.Logger interface
func (w *LoggerWrapper) Log(ctx context.Context, level stdslog.Level, msg string, args ...any) {
	logLevel := FromSlogLevel(level)
	w.outputWithLevel(logLevel, ctx, msg, args...)
}

// LogAttrs implements slog.Logger interface with slog.Attr support
func (w *LoggerWrapper) LogAttrs(ctx context.Context, level stdslog.Level, msg string, attrs ...stdslog.Attr) {
	logLevel := FromSlogLevel(level)
	args := make([]any, 0, len(attrs)*2)
	for _, attr := range attrs {
		args = append(args, attr.Key, attr.Value)
	}
	w.outputWithLevel(logLevel, ctx, msg, args...)
}

// DebugCtx logs at debug level with context
func (w *LoggerWrapper) DebugCtx(ctx context.Context, msg string, args ...any) {
	w.outputWithLevel(log.DebugLevel, ctx, msg, args...)
}

// InfoCtx logs at info level with context
func (w *LoggerWrapper) InfoCtx(ctx context.Context, msg string, args ...any) {
	w.outputWithLevel(log.InfoLevel, ctx, msg, args...)
}

// WarnCtx logs at warn level with context
func (w *LoggerWrapper) WarnCtx(ctx context.Context, msg string, args ...any) {
	w.outputWithLevel(log.WarnLevel, ctx, msg, args...)
}

// ErrorCtx logs at error level with context
func (w *LoggerWrapper) ErrorCtx(ctx context.Context, msg string, args ...any) {
	w.outputWithLevel(log.ErrorLevel, ctx, msg, args...)
}

// Original log.Logger interface delegation

// Trace implements log.Logger
//
//nolint:goprintffuncname // Keeping consistent with log.Logger interface
func (w *LoggerWrapper) Trace(format string, args ...any) {
	w.logger.Trace(format, args...)
}

// Debug implements log.Logger
func (w *LoggerWrapper) Debug(format string, args ...any) {
	w.logger.Debug(format, args...)
}

// Info implements log.Logger
func (w *LoggerWrapper) Info(format string, args ...any) {
	w.logger.Info(format, args...)
}

// Warn implements log.Logger
func (w *LoggerWrapper) Warn(format string, args ...any) {
	w.logger.Warn(format, args...)
}

// Error implements log.Logger
func (w *LoggerWrapper) Error(format string, args ...any) {
	w.logger.Error(format, args...)
}

// Fatal implements log.Logger
func (w *LoggerWrapper) Fatal(format string, args ...any) {
	w.logger.Fatal(format, args...)
}

// Panic implements log.Logger
func (w *LoggerWrapper) Panic(format string, args ...any) {
	w.logger.Panic(format, args...)
}

// Context-aware methods from original log.Logger

// CtxTrace implements log.Logger
func (w *LoggerWrapper) CtxTrace(ctx context.Context, format string, args ...any) {
	w.logger.CtxTrace(ctx, format, args...)
}

// CtxDebug implements log.Logger
func (w *LoggerWrapper) CtxDebug(ctx context.Context, format string, args ...any) {
	w.logger.CtxDebug(ctx, format, args...)
}

// CtxInfo implements log.Logger
func (w *LoggerWrapper) CtxInfo(ctx context.Context, format string, args ...any) {
	w.logger.CtxInfo(ctx, format, args...)
}

// CtxWarn implements log.Logger
func (w *LoggerWrapper) CtxWarn(ctx context.Context, format string, args ...any) {
	w.logger.CtxWarn(ctx, format, args...)
}

// CtxError implements log.Logger
func (w *LoggerWrapper) CtxError(ctx context.Context, format string, args ...any) {
	w.logger.CtxError(ctx, format, args...)
}

// CtxFatal implements log.Logger
func (w *LoggerWrapper) CtxFatal(ctx context.Context, format string, args ...any) {
	w.logger.CtxFatal(ctx, format, args...)
}

// CtxPanic implements log.Logger
func (w *LoggerWrapper) CtxPanic(ctx context.Context, format string, args ...any) {
	w.logger.CtxPanic(ctx, format, args...)
}

// With implements log.Logger
func (w *LoggerWrapper) With(args ...any) log.Logger {
	return &LoggerWrapper{logger: w.logger.With(args...)}
}

// WithGroup implements log.Logger
func (w *LoggerWrapper) WithGroup(name string) log.Logger {
	return &LoggerWrapper{logger: w.logger.WithGroup(name)}
}

// SetLevel implements log.Logger
func (w *LoggerWrapper) SetLevel(level log.Level) {
	w.logger.SetLevel(level)
}

// SetOutput implements log.Logger
func (w *LoggerWrapper) SetOutput(writer io.Writer) {
	w.logger.SetOutput(writer)
}

// AddOutputs implements log.Logger
func (w *LoggerWrapper) AddOutputs(writers ...io.Writer) {
	w.logger.AddOutputs(writers...)
}

// Flush implements log.Logger
func (w *LoggerWrapper) Flush() {
	w.logger.Flush()
}

// Close implements log.Logger
func (w *LoggerWrapper) Close() error {
	return w.logger.Close()
}

// GetLogger returns the underlying log.Logger
func (w *LoggerWrapper) GetLogger() log.Logger {
	return w.logger
}

// Helper method for consistent output
func (w *LoggerWrapper) outputWithLevel(level log.Level, ctx context.Context, msg string, args ...any) {
	switch level {
	case log.TraceLevel:
		w.logger.CtxTrace(ctx, msg, args...)
	case log.DebugLevel:
		w.logger.CtxDebug(ctx, msg, args...)
	case log.InfoLevel:
		w.logger.CtxInfo(ctx, msg, args...)
	case log.WarnLevel:
		w.logger.CtxWarn(ctx, msg, args...)
	case log.ErrorLevel:
		w.logger.CtxError(ctx, msg, args...)
	case log.FatalLevel:
		w.logger.CtxFatal(ctx, msg, args...)
	case log.PanicLevel:
		w.logger.CtxPanic(ctx, msg, args...)
	default:
		w.logger.CtxInfo(ctx, msg, args...)
	}
}
