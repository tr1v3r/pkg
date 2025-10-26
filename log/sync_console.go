package log

import (
	"context"
	"io"
	"os"
	"sync"
)

// SyncConsoleHandler is a synchronous version of ConsoleHandler for testing
// It writes directly to the output without async buffering
type SyncConsoleHandler struct {
	formatter Formatter
	level     Level
	out       io.Writer
	mu        sync.RWMutex
}

// NewSyncConsoleHandler creates a new synchronous console handler
func NewSyncConsoleHandler(level Level) *SyncConsoleHandler {
	return &SyncConsoleHandler{
		formatter: NewTextFormatter(false), // No color for tests
		level:     level,
		out:       os.Stdout,
	}
}

// SetLevel sets the minimum log level for this handler
func (h *SyncConsoleHandler) SetLevel(level Level) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.level = level
}

// SetOutput sets the output writer for this handler
func (h *SyncConsoleHandler) SetOutput(w io.Writer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.out = w
}

// AddOutput adds an additional output writer to this handler
func (h *SyncConsoleHandler) AddOutput(w io.Writer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.out = io.MultiWriter(h.out, w)
}

// Output writes a log message to the console synchronously
func (h *SyncConsoleHandler) Output(level Level, ctx context.Context, format string, args ...interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if level < h.level {
		return
	}

	// Format the message
	msg := h.formatter.Format(level, ctx, format, args...)

	// Write directly to output
	_, _ = h.out.Write([]byte(msg))
}

// Write implements io.Writer for direct writes
func (h *SyncConsoleHandler) Write(p []byte) (int, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.out.Write(p)
}

// Flush is a no-op for synchronous handler
func (h *SyncConsoleHandler) Flush() {
	// No-op for synchronous handler
}

// Close is a no-op for synchronous handler
func (h *SyncConsoleHandler) Close() error {
	return nil
}