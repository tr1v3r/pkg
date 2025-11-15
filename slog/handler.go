package slog

import (
	"context"
	"io"

	"github.com/tr1v3r/pkg/log"
	stdslog "log/slog"
)

// ExtendedHandler extends the existing log.Handler interface to support slog compatibility
type ExtendedHandler interface {
	log.Handler

	// slog.Handler compatibility methods
	HandleSlog(ctx context.Context, record stdslog.Record) error
	Enabled(ctx context.Context, level stdslog.Level) bool
	WithSlogAttrs(attrs []stdslog.Attr) ExtendedHandler
	WithSlogGroup(name string) ExtendedHandler
}

// SlogHandlerAdapter wraps a log.Handler to implement stdslog.Handler interface
type SlogHandlerAdapter struct {
	handler log.Handler
}

// NewSlogHandlerAdapter creates a new adapter that wraps a log.Handler
func NewSlogHandlerAdapter(handler log.Handler) *SlogHandlerAdapter {
	return &SlogHandlerAdapter{
		handler: handler,
	}
}

// Enabled implements stdslog.Handler
func (a *SlogHandlerAdapter) Enabled(ctx context.Context, level stdslog.Level) bool {
	// Convert slog level to log package level and check if enabled
	logLevel := FromSlogLevel(level)

	// Get the current level from the handler if possible
	if leveledHandler, ok := a.handler.(interface{ GetLevel() log.Level }); ok {
		currentLevel := leveledHandler.GetLevel()
		return logLevel >= currentLevel
	}

	// Default to Info level if we can't determine current level
	return logLevel >= log.InfoLevel
}

// Handle implements stdslog.Handler
func (a *SlogHandlerAdapter) Handle(ctx context.Context, record stdslog.Record) error {
	// Convert slog record to log package format
	logLevel := FromSlogLevel(record.Level)

	// Convert attributes to args
	var args []any
	record.Attrs(func(attr stdslog.Attr) bool {
		args = append(args, attr.Key, attr.Value)
		return true
	})

	// Output using the underlying handler
	a.handler.Output(logLevel, ctx, record.Message, args...)
	return nil
}

// WithAttrs implements stdslog.Handler
func (a *SlogHandlerAdapter) WithAttrs(attrs []stdslog.Attr) stdslog.Handler {
	// Convert slog attributes to log package args
	args := make([]any, 0, len(attrs)*2)
	for _, attr := range attrs {
		args = append(args, attr.Key, attr.Value)
	}

	// Create new handler with attributes
	newHandler := a.handler.With(args...)
	return NewSlogHandlerAdapter(newHandler)
}

// WithGroup implements stdslog.Handler
func (a *SlogHandlerAdapter) WithGroup(name string) stdslog.Handler {
	// Create new handler with group
	newHandler := a.handler.WithGroup(name)
	return NewSlogHandlerAdapter(newHandler)
}

// Write implements io.Writer for compatibility with log.Handler
func (a *SlogHandlerAdapter) Write(p []byte) (n int, err error) {
	if writer, ok := a.handler.(io.Writer); ok {
		return writer.Write(p)
	}
	return len(p), nil
}

// GetHandler returns the underlying log.Handler
func (a *SlogHandlerAdapter) GetHandler() log.Handler {
	return a.handler
}
