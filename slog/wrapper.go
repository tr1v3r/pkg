package slog

import (
	"context"

	"github.com/tr1v3r/pkg/log"
	stdslog "log/slog"
)

// AsSlogLogger converts an existing log.Logger to a stdslog.Logger
// This allows seamless integration with existing code that uses stdslog.Logger
func AsSlogLogger(logger log.Logger) *stdslog.Logger {
	// We need to get the underlying handler from the logger
	// For now, create a simple adapter
	return stdslog.New(&LoggerToSlogAdapter{logger: logger})
}

// AsExtendedLogger converts an existing log.Logger to an ExtendedLogger with slog compatibility
func AsExtendedLogger(logger log.Logger) ExtendedLogger {
	return &LoggerWrapper{logger: logger}
}

// NewFromLogger creates a new stdslog.Logger from an existing log.Logger
// This is a convenience function that wraps AsSlogLogger
func NewFromLogger(logger log.Logger) *stdslog.Logger {
	return AsSlogLogger(logger)
}

// WrapHandler wraps a log.Handler to implement stdslog.Handler interface
func WrapHandler(handler log.Handler) stdslog.Handler {
	return NewSlogHandlerAdapter(handler)
}

// WrapHandlers wraps multiple log.Handlers and returns a single stdslog.Handler
// This uses a multi-handler pattern if needed
func WrapHandlers(handlers ...log.Handler) stdslog.Handler {
	if len(handlers) == 1 {
		return NewSlogHandlerAdapter(handlers[0])
	}

	// If multiple handlers, we need to create a multi-handler adapter
	return &MultiHandlerAdapter{
		handlers: handlers,
	}
}

// MultiHandlerAdapter implements stdslog.Handler for multiple log.Handler instances
type MultiHandlerAdapter struct {
	handlers []log.Handler
}

// Enabled implements stdslog.Handler
func (m *MultiHandlerAdapter) Enabled(ctx context.Context, level stdslog.Level) bool {
	// Return true if any handler is enabled for this level
	for _, handler := range m.handlers {
		adapter := NewSlogHandlerAdapter(handler)
		if adapter.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle implements stdslog.Handler
func (m *MultiHandlerAdapter) Handle(ctx context.Context, record stdslog.Record) error {
	// Send to all handlers, collect any errors
	for _, handler := range m.handlers {
		adapter := NewSlogHandlerAdapter(handler)
		if err := adapter.Handle(ctx, record); err != nil {
			// For logging, we typically don't want to fail completely
			// But we could collect errors if needed
			continue
		}
	}
	return nil
}

// WithAttrs implements stdslog.Handler
func (m *MultiHandlerAdapter) WithAttrs(attrs []stdslog.Attr) stdslog.Handler {
	newHandlers := make([]log.Handler, len(m.handlers))
	for i, handler := range m.handlers {
		adapter := NewSlogHandlerAdapter(handler)
		newHandlers[i] = adapter.WithAttrs(attrs).(*SlogHandlerAdapter).GetHandler()
	}
	return &MultiHandlerAdapter{handlers: newHandlers}
}

// WithGroup implements stdslog.Handler
func (m *MultiHandlerAdapter) WithGroup(name string) stdslog.Handler {
	newHandlers := make([]log.Handler, len(m.handlers))
	for i, handler := range m.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}
	return &MultiHandlerAdapter{handlers: newHandlers}
}

// LoggerToSlogAdapter adapts a log.Logger to implement stdslog.Handler
type LoggerToSlogAdapter struct {
	logger log.Logger
}

// Enabled implements stdslog.Handler
func (a *LoggerToSlogAdapter) Enabled(ctx context.Context, level stdslog.Level) bool {
	logLevel := FromSlogLevel(level)
	// Default to Info level if we can't determine current level
	return logLevel >= log.InfoLevel
}

// Handle implements stdslog.Handler
func (a *LoggerToSlogAdapter) Handle(ctx context.Context, record stdslog.Record) error {
	logLevel := FromSlogLevel(record.Level)

	// Convert attributes to args
	args := make([]any, 0, 8) // Pre-allocate reasonable capacity
	record.Attrs(func(attr stdslog.Attr) bool {
		args = append(args, attr.Key, attr.Value)
		return true
	})

	// Use the logger's context methods if context is provided
	if ctx != nil {
		switch logLevel {
		case log.TraceLevel:
			a.logger.CtxTrace(ctx, record.Message, args...)
		case log.DebugLevel:
			a.logger.CtxDebug(ctx, record.Message, args...)
		case log.InfoLevel:
			a.logger.CtxInfo(ctx, record.Message, args...)
		case log.WarnLevel:
			a.logger.CtxWarn(ctx, record.Message, args...)
		case log.ErrorLevel:
			a.logger.CtxError(ctx, record.Message, args...)
		case log.FatalLevel:
			a.logger.CtxFatal(ctx, record.Message, args...)
		case log.PanicLevel:
			a.logger.CtxPanic(ctx, record.Message, args...)
		default:
			a.logger.CtxInfo(ctx, record.Message, args...)
		}
	} else {
		// Use regular methods if no context
		switch logLevel {
		case log.TraceLevel:
			a.logger.Trace(record.Message, args...)
		case log.DebugLevel:
			a.logger.Debug(record.Message, args...)
		case log.InfoLevel:
			a.logger.Info(record.Message, args...)
		case log.WarnLevel:
			a.logger.Warn(record.Message, args...)
		case log.ErrorLevel:
			a.logger.Error(record.Message, args...)
		case log.FatalLevel:
			a.logger.Fatal(record.Message, args...)
		case log.PanicLevel:
			a.logger.Panic(record.Message, args...)
		default:
			a.logger.Info(record.Message, args...)
		}
	}

	return nil
}

// WithAttrs implements stdslog.Handler
func (a *LoggerToSlogAdapter) WithAttrs(attrs []stdslog.Attr) stdslog.Handler {
	// Convert slog attributes to log package args
	args := make([]any, 0, len(attrs)*2)
	for _, attr := range attrs {
		args = append(args, attr.Key, attr.Value)
	}

	// Create new logger with attributes
	newLogger := a.logger.With(args...)
	return &LoggerToSlogAdapter{logger: newLogger}
}

// WithGroup implements stdslog.Handler
func (a *LoggerToSlogAdapter) WithGroup(name string) stdslog.Handler {
	// Create new logger with group
	newLogger := a.logger.WithGroup(name)
	return &LoggerToSlogAdapter{logger: newLogger}
}
