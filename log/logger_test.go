package log

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	// Test with no handlers (should use default console handler)
	logger := NewLogger()
	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}

	// Test with custom handlers
	buf := &bytes.Buffer{}
	handler := NewSyncConsoleHandler(InfoLevel)
	handler.SetOutput(buf)

	logger = NewLogger(handler)
	if logger == nil {
		t.Fatal("NewLogger(handler) returned nil")
	}

	// Test logging works
	logger.Info("test message")
	logger.Flush()

	if !strings.Contains(buf.String(), "test message") {
		t.Errorf("Expected log output to contain 'test message', got: %s", buf.String())
	}
}

func TestBaseLogger_SetLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewSyncConsoleHandler(InfoLevel)
	handler.SetOutput(buf)

	logger := NewLogger(handler).(*baseLogger)

	// Test setting level
	logger.SetLevel(DebugLevel)

	// Verify debug messages are now logged
	logger.Debug("debug message")
	logger.Flush()

	if !strings.Contains(buf.String(), "debug message") {
		t.Errorf("Expected debug message to be logged after setting DebugLevel, got: %s", buf.String())
	}

	// Test setting higher level
	buf.Reset()
	logger.SetLevel(WarnLevel)
	logger.Debug("debug message 2")
	logger.Flush()

	if strings.Contains(buf.String(), "debug message 2") {
		t.Errorf("Expected debug message to be filtered out after setting WarnLevel, got: %s", buf.String())
	}
}

func TestBaseLogger_SetOutput(t *testing.T) {
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}
	handler := NewSyncConsoleHandler(InfoLevel)
	handler.SetOutput(buf1)

	logger := NewLogger(handler)

	// Test SetOutput
	logger.SetOutput(buf2)
	logger.Info("test output")
	logger.Flush()

	if buf1.String() != "" {
		t.Errorf("Expected original buffer to be empty after SetOutput, got: %s", buf1.String())
	}
	if !strings.Contains(buf2.String(), "test output") {
		t.Errorf("Expected new buffer to contain log, got: %s", buf2.String())
	}
}

func TestBaseLogger_AddOutput(t *testing.T) {
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}
	handler := NewSyncConsoleHandler(InfoLevel)
	handler.SetOutput(buf1)

	logger := NewLogger(handler)

	// Test AddOutput
	logger.AddOutput(buf2)
	logger.Info("test multi-output")
	logger.Flush()

	if !strings.Contains(buf1.String(), "test multi-output") {
		t.Errorf("Expected buffer1 to contain log, got: %s", buf1.String())
	}
	if !strings.Contains(buf2.String(), "test multi-output") {
		t.Errorf("Expected buffer2 to contain log, got: %s", buf2.String())
	}
}

func TestBaseLogger_Flush(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewSyncConsoleHandler(InfoLevel)
	handler.SetOutput(buf)

	logger := NewLogger(handler)

	// Test Flush doesn't panic
	logger.Flush()

	// Test Flush after logging
	logger.Info("flush test")
	logger.Flush()

	if !strings.Contains(buf.String(), "flush test") {
		t.Errorf("Expected flush to write logs, got: %s", buf.String())
	}
}

func TestBaseLogger_Close(t *testing.T) {
	handler := NewSyncConsoleHandler(InfoLevel)
	logger := NewLogger(handler)

	// Test Close doesn't panic
	err := logger.Close()
	if err != nil {
		t.Errorf("Expected Close to return nil error, got: %v", err)
	}

	// Test Close with multiple handlers
	handler1 := NewSyncConsoleHandler(InfoLevel)
	handler2 := NewSyncConsoleHandler(InfoLevel)
	logger = NewLogger(handler1, handler2)

	err = logger.Close()
	if err != nil {
		t.Errorf("Expected Close with multiple handlers to return nil error, got: %v", err)
	}
}

func TestBaseLogger_With(t *testing.T) {
	logger := NewLogger()

	// Test With returns the same logger (baseLogger doesn't support structured logging)
	newLogger := logger.With("key", "value")
	if newLogger != logger {
		t.Error("Expected baseLogger.With() to return the same logger")
	}
}

func TestBaseLogger_WithGroup(t *testing.T) {
	logger := NewLogger()

	// Test WithGroup returns the same logger (baseLogger doesn't support structured logging)
	newLogger := logger.WithGroup("testgroup")
	if newLogger != logger {
		t.Error("Expected baseLogger.WithGroup() to return the same logger")
	}
}

func TestBaseLogger_LoggingMethods(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewSyncConsoleHandler(TraceLevel) // Set to trace to capture all levels
	handler.SetOutput(buf)

	logger := NewLogger(handler)

	// Test all logging levels
	testCases := []struct {
		name   string
		logFn  func(string, ...any)
		ctxLogFn func(context.Context, string, ...any)
		expected string
	}{
		{"Trace", logger.Trace, logger.CtxTrace, "trace message"},
		{"Debug", logger.Debug, logger.CtxDebug, "debug message"},
		{"Info", logger.Info, logger.CtxInfo, "info message"},
		{"Warn", logger.Warn, logger.CtxWarn, "warn message"},
		{"Error", logger.Error, logger.CtxError, "error message"},
		{"Fatal", logger.Fatal, logger.CtxFatal, "fatal message"},
		{"Panic", logger.Panic, logger.CtxPanic, "panic message"},
	}

	for _, tc := range testCases {
		buf.Reset()

		// Test without context
		tc.logFn(tc.expected)
		logger.Flush()

		if !strings.Contains(buf.String(), tc.expected) {
			t.Errorf("%s: expected log to contain '%s', got: %s", tc.name, tc.expected, buf.String())
		}

		// Test with context
		buf.Reset()
		ctx := context.WithValue(context.Background(), LogIDKey, "test-123")
		tc.ctxLogFn(ctx, "ctx "+tc.expected)
		logger.Flush()

		if !strings.Contains(buf.String(), "ctx "+tc.expected) {
			t.Errorf("%s with context: expected log to contain 'ctx %s', got: %s", tc.name, tc.expected, buf.String())
		}
	}
}

func TestBaseLogger_Output(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := NewSyncConsoleHandler(InfoLevel)
	handler.SetOutput(buf)

	logger := NewLogger(handler).(*baseLogger)

	// Test output method directly
	logger.output(InfoLevel, nil, "direct output test")
	logger.Flush()

	if !strings.Contains(buf.String(), "direct output test") {
		t.Errorf("Expected direct output to be logged, got: %s", buf.String())
	}

	// Test output with context
	buf.Reset()
	ctx := context.WithValue(context.Background(), LogIDKey, "test-456")
	logger.output(InfoLevel, ctx, "context output test")
	logger.Flush()

	if !strings.Contains(buf.String(), "context output test") {
		t.Errorf("Expected context output to be logged, got: %s", buf.String())
	}
}

func TestMultiError(t *testing.T) {
	// Test MultiError with no errors
	multiErr := &MultiError{Errors: []error{}}
	if multiErr.Error() != "no errors" {
		t.Errorf("Expected 'no errors', got: %s", multiErr.Error())
	}

	// Test MultiError with multiple errors
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	multiErr = &MultiError{Errors: []error{err1, err2}}

	expected := "multiple errors occurred while closing handlers"
	if multiErr.Error() != expected {
		t.Errorf("Expected '%s', got: %s", expected, multiErr.Error())
	}
}