package log

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ConsoleHandler writes log messages to the console with async buffering
type ConsoleHandler struct {
	formatter Formatter
	level     Level
	ch        chan []byte
	out       io.Writer
	mu        sync.RWMutex

	closeOnce sync.Once
	closed    chan struct{}
	once      sync.Once
}

// NewConsoleHandler creates a new console handler
func NewConsoleHandler(level Level) *ConsoleHandler {
	return &ConsoleHandler{
		formatter: NewTextFormatter(true), // Enable color by default
		level:     level,
		ch:        make(chan []byte, 8192), // 8KB buffer
		out:       os.Stdout,
		closed:    make(chan struct{}),
	}
}

// SetLevel sets the minimum log level for this handler
func (h *ConsoleHandler) SetLevel(level Level) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.level = level
}

// SetOutput sets the output writer for this handler
func (h *ConsoleHandler) SetOutput(w io.Writer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.out = w
}

// AddOutputs adds multiple output writers to this handler
func (h *ConsoleHandler) AddOutputs(writers ...io.Writer) {
	// Fast path: no writers provided
	if len(writers) == 0 {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Combine existing output with new writers
	allWriters := []io.Writer{h.out}
	allWriters = append(allWriters, writers...)
	h.out = io.MultiWriter(allWriters...)
}

// Output writes a log message to the console
func (h *ConsoleHandler) Output(level Level, ctx context.Context, format string, args ...any) {
	h.once.Do(func() { go h.serve() })

	if level < h.level {
		return
	}

	// Format the message
	msg := h.formatter.Format(level, ctx, format, args...)

	// Send to async channel
	select {
	case h.ch <- []byte(msg):
		// Message queued successfully
	case <-h.closed:
		// Handler is closed, drop message
	}
}

// Write implements io.Writer for direct writes
func (h *ConsoleHandler) Write(p []byte) (int, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.out.Write(p)
}

// Flush ensures all buffered messages are written
func (h *ConsoleHandler) Flush() {
	// Give other goroutines a chance to run
	runtime.Gosched()

	// Process as many messages as possible from the channel
	for i := 0; i < cap(h.ch); i++ {
		select {
		case msg, ok := <-h.ch:
			if !ok {
				h.close() // Channel closed, handler is shutting down
				return
			}
			if _, err := h.Write(msg); err != nil {
				// Log error to stderr since our handler might be failing
				fmt.Fprintf(os.Stderr, "console handler write failed: %v\n", err)
			}
		default:
			return // No more messages in channel
		}
	}
}

// Close shuts down the handler and releases resources
func (h *ConsoleHandler) Close() error {
	close(h.ch)
	h.Flush()
	<-h.closed
	return nil
}

// serve processes messages from the async channel
func (h *ConsoleHandler) serve() {
	for msg := range h.ch {
		if _, err := h.Write(msg); err != nil {
			fmt.Fprintf(os.Stderr, "console handler write failed: %v\n", err)
		}
	}
	h.close()
}

// close marks the handler as closed
func (h *ConsoleHandler) close() {
	h.closeOnce.Do(func() {
		close(h.closed)
	})
}

// TextFormatter formats log messages as human-readable text
type TextFormatter struct {
	color bool
}

// NewTextFormatter creates a new text formatter
func NewTextFormatter(color bool) *TextFormatter {
	return &TextFormatter{color: color}
}

// Format converts log data into a formatted string
func (f *TextFormatter) Format(level Level, ctx context.Context, format string, args ...any) string {
	var buf strings.Builder

	// Add color if enabled
	if f.color {
		buf.WriteString("\033[38;5;")
		buf.WriteString(fmt.Sprint(level.Color()))
		buf.WriteString("m")
	}

	// Timestamp
	buf.WriteString(time.Now().Format(time.RFC3339))
	buf.WriteByte(' ')

	// Log level
	buf.WriteByte('[')
	buf.WriteString(level.String())
	buf.WriteByte(']')
	buf.WriteByte(' ')

	// Context data (e.g., log_id)
	if ctx != nil {
		if logID := f.getLogID(ctx); logID != "" {
			buf.WriteString(logID)
			buf.WriteByte(' ')
		}
	}

	// Message
	if len(args) > 0 {
		buf.WriteString(fmt.Sprintf(format, args...))
	} else {
		buf.WriteString(format)
	}

	// Reset color if enabled
	if f.color {
		buf.WriteString("\033[0m")
	}

	buf.WriteByte('\n')

	return buf.String()
}

// getLogID extracts log_id from context
func (f *TextFormatter) getLogID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(LogIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}