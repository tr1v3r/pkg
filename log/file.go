package log

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// RotationInterval defines when to rotate log files
type RotationInterval int

const (
	// RotationNone means no rotation (single file)
	RotationNone RotationInterval = iota
	// RotationHourly rotates logs every hour
	RotationHourly
	// RotationDaily rotates logs every day
	RotationDaily
	// RotationWeekly rotates logs every week
	RotationWeekly
	// RotationMonthly rotates logs every month
	RotationMonthly
)

// FileHandler writes log messages to files with rotation support
type FileHandler struct {
	formatter Formatter
	level     Level
	out       io.Writer
	mu        sync.RWMutex

	// File configuration
	logDir       string
	filename     string
	rotation     RotationInterval
	rotationTime time.Time
	filePath     string

	// Async buffering
	ch        chan []byte
	closeOnce sync.Once
	closed    chan struct{}
	once      sync.Once

	// Performance optimization: pre-calculated next rotation time
	nextRotationTime time.Time
}

// FileHandlerConfig holds configuration for FileHandler
type FileHandlerConfig struct {
	// LogDir is the directory where log files will be stored
	LogDir string
	// Filename is the base filename pattern (can include strftime-like placeholders)
	Filename string
	// Rotation defines when to rotate log files
	Rotation RotationInterval
	// Level is the minimum log level to output
	Level Level
}

// NewFileHandler creates a new file handler with the specified configuration
func NewFileHandler(config FileHandlerConfig) (*FileHandler, error) {
	// Set defaults
	if config.LogDir == "" {
		config.LogDir = "./logs"
	}
	if config.Filename == "" {
		config.Filename = "app.log"
	}
	if config.Level == 0 {
		config.Level = InfoLevel
	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(config.LogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	handler := &FileHandler{
		formatter: NewTextFormatter(false), // No color for files
		level:     config.Level,
		logDir:    config.LogDir,
		filename:  config.Filename,
		rotation:  config.Rotation,
		ch:        make(chan []byte, 8192), // 8KB buffer
		closed:    make(chan struct{}),
	}

	// Initialize the first log file
	if err := handler.tryRotate(time.Now()); err != nil {
		return nil, fmt.Errorf("failed to initialize log file: %w", err)
	}

	return handler, nil
}

// SetLevel sets the minimum log level for this handler
func (h *FileHandler) SetLevel(level Level) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.level = level
}

// SetOutput sets the output writer for this handler
func (h *FileHandler) SetOutput(w io.Writer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.out = w
}

// AddOutputs adds multiple output writers to this handler
func (h *FileHandler) AddOutputs(writers ...io.Writer) {
	// Fast path: no writers provided
	if len(writers) == 0 {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Combine existing output with new writers
	h.out = io.MultiWriter(append(append(make([]io.Writer, 0, len(writers)+1), h.out), writers...)...)
}

// Output writes a log message to the file
func (h *FileHandler) Output(level Level, ctx context.Context, format string, args ...any) {
	h.once.Do(func() { go h.serve() })

	if level < h.level {
		return
	}

	// Format the message (this is the only synchronous work)
	msg := h.formatter.Format(level, ctx, format, args...)

	// Send to async channel immediately - rotation check happens during consumption
	select {
	case h.ch <- []byte(msg):
		// Message queued successfully
	case <-h.closed:
		// Handler is closed, drop message
	}
}

// Write implements io.Writer for direct writes
func (h *FileHandler) Write(p []byte) (int, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.out == nil {
		return 0, fmt.Errorf("file handler output is not set")
	}
	return h.out.Write(p)
}

// Flush ensures all buffered messages are written
func (h *FileHandler) Flush() {
	// Process as many messages as possible from the channel
	for i := 0; i < cap(h.ch); i++ {
		select {
		case msg, ok := <-h.ch:
			if !ok {
				h.close() // Channel closed, handler is shutting down
				return
			}
			if _, err := h.Write(msg); err != nil {
				fmt.Fprintf(os.Stderr, "file handler write failed: %v\n", err)
			}
		default:
			return // No more messages in channel
		}
	}
}

// Close shuts down the handler and releases resources
func (h *FileHandler) Close() error {
	close(h.ch)
	h.Flush()
	<-h.closed

	if h.out != nil {
		if file, ok := h.out.(*os.File); ok {
			return file.Close()
		}
	}
	return nil
}

// serve processes messages from the async channel
func (h *FileHandler) serve() {
	for msg := range h.ch {
		// Check rotation before writing each message batch
		if err := h.tryRotate(time.Now()); err != nil {
			fmt.Fprintf(os.Stderr, "file handler rotation failed: %v\n", err)
			// Continue processing messages even if rotation fails
		}

		if _, err := h.Write(msg); err != nil {
			fmt.Fprintf(os.Stderr, "file handler write failed: %v\n", err)
		}
	}
	h.close()
}

// close marks the handler as closed
func (h *FileHandler) close() {
	h.closeOnce.Do(func() {
		close(h.closed)
	})
}

// tryRotate attempts to rotate the log file if needed based on the current time.
// It handles both initial file setup and time-based rotation.
// Returns nil if no rotation was needed or if rotation succeeded.
func (h *FileHandler) tryRotate(now time.Time) error {
	// Fast path: handle file initialization or no rotation configuration
	if h.rotation == RotationNone || h.rotationTime.IsZero() {
		// No rotation configured or first time setup - just ensure file exists
		return h.openFileIfNull(now)
	}

	// Check if current time has passed the pre-calculated rotation boundary
	if now.After(h.nextRotationTime) {
		return h.rotate(now)
	}

	return nil
}

// calculateNextRotationTime calculates when the next rotation should occur
func (h *FileHandler) calculateNextRotationTime(currentTime time.Time) time.Time {
	switch h.rotation {
	case RotationHourly:
		// Next rotation at start of next hour
		return currentTime.Truncate(time.Hour).Add(time.Hour)
	case RotationDaily:
		// Next rotation at start of next day
		return currentTime.Truncate(24 * time.Hour).Add(24 * time.Hour)
	case RotationWeekly:
		// Next rotation at start of next week (Monday 00:00)
		weekStart := startOfWeek(currentTime)
		return weekStart.Add(7 * 24 * time.Hour)
	case RotationMonthly:
		// Next rotation at start of next month
		monthStart := startOfMonth(currentTime)
		return monthStart.AddDate(0, 1, 0)
	default:
		// No rotation - return far future
		return currentTime.AddDate(100, 0, 0)
	}
}

// rotate performs the actual file rotation
func (h *FileHandler) rotate(now time.Time) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Close current file if open
	if h.out != nil {
		if file, ok := h.out.(*os.File); ok {
			if err := file.Close(); err != nil {
				return fmt.Errorf("failed to close current log file: %w", err)
			}
		}
	}

	// Open new file
	if err := h.openFile(now); err != nil {
		return err
	}

	// Calculate next rotation time
	h.nextRotationTime = h.calculateNextRotationTime(now)

	return nil
}

// openFileIfNull opens a new log file only if out is nil
func (h *FileHandler) openFileIfNull(now time.Time) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Double-check under lock
	if h.out != nil {
		return nil
	}

	return h.openFile(now)
}

// openFile opens a new log file based on the current time
// Caller must hold the lock
func (h *FileHandler) openFile(now time.Time) error {
	// Generate filename with timestamp if needed
	filename := h.formatFilename(now)
	filePath := filepath.Join(h.logDir, filename)

	// Open file for writing (append mode)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	h.filePath = filePath
	h.out = file
	h.rotationTime = now

	// Calculate initial next rotation time
	h.nextRotationTime = h.calculateNextRotationTime(now)

	return nil
}

// formatFilename generates the filename with appropriate timestamp
func (h *FileHandler) formatFilename(now time.Time) string {
	if h.rotation == RotationNone {
		return h.filename
	}

	// Extract file extension
	ext := filepath.Ext(h.filename)
	base := strings.TrimSuffix(h.filename, ext)

	// Add timestamp based on rotation interval
	var timestamp string
	switch h.rotation {
	case RotationHourly:
		timestamp = now.Format("2006-01-02_15")
	case RotationDaily:
		timestamp = now.Format("2006-01-02")
	case RotationWeekly:
		// Use year-week format
		year, week := now.ISOWeek()
		timestamp = fmt.Sprintf("%d-W%02d", year, week)
	case RotationMonthly:
		timestamp = now.Format("2006-01")
	default:
		timestamp = now.Format("2006-01-02_15-04-05")
	}

	return fmt.Sprintf("%s_%s%s", base, timestamp, ext)
}

// startOfWeek returns the start of the week (Monday 00:00)
func startOfWeek(t time.Time) time.Time {
	// Go's time package considers Monday as the first day of the week
	weekday := t.Weekday()
	if weekday == time.Sunday {
		// For ISO week, if it's Sunday, we need to go back 6 days to get to Monday
		weekday = 7
	}
	// Calculate days to subtract to get to Monday
	daysToSubtract := int(weekday - time.Monday)
	return time.Date(t.Year(), t.Month(), t.Day()-daysToSubtract, 0, 0, 0, 0, t.Location())
}

// startOfMonth returns the start of the month (1st 00:00)
func startOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// GetCurrentFilePath returns the current log file path
func (h *FileHandler) GetCurrentFilePath() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.filePath
}

