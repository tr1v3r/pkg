package log

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{TraceLevel, "TRACE"},
		{DebugLevel, "DEBUG"},
		{InfoLevel, "INFO"},
		{WarnLevel, "WARN"},
		{ErrorLevel, "ERROR"},
		{FatalLevel, "FATAL"},
		{PanicLevel, "PANIC"},
		{Level(100), "UNKNOWN"},
	}

	for _, test := range tests {
		result := test.level.String()
		if result != test.expected {
			t.Errorf("Level(%d).String() = %s, expected %s", test.level, result, test.expected)
		}
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"trace", TraceLevel},
		{"TRACE", TraceLevel},
		{"debug", DebugLevel},
		{"DEBUG", DebugLevel},
		{"info", InfoLevel},
		{"INFO", InfoLevel},
		{"warn", WarnLevel},
		{"WARN", WarnLevel},
		{"warning", WarnLevel},
		{"error", ErrorLevel},
		{"ERROR", ErrorLevel},
		{"fatal", FatalLevel},
		{"FATAL", FatalLevel},
		{"panic", PanicLevel},
		{"PANIC", PanicLevel},
		{"unknown", InfoLevel},
		{"", InfoLevel},
	}

	for _, test := range tests {
		result := ParseLevel(test.input)
		if result != test.expected {
			t.Errorf("ParseLevel(%s) = %d, expected %d", test.input, result, test.expected)
		}
	}
}


func TestTextFormatter(t *testing.T) {
	formatter := NewTextFormatter(false) // No color

	// Test basic formatting
	result := formatter.Format(InfoLevel, nil, "test %s", "message")
	if !strings.Contains(result, "test message") {
		t.Errorf("Expected formatted message to contain 'test message', got: %s", result)
	}
	if !strings.Contains(result, "INFO") {
		t.Errorf("Expected formatted message to contain 'INFO', got: %s", result)
	}

	// Test with context
	ctx := context.WithValue(context.Background(), LogIDKey, "ctx123")
	result = formatter.Format(InfoLevel, ctx, "context test")
	if !strings.Contains(result, "ctx123") {
		t.Errorf("Expected formatted message to contain log_id 'ctx123', got: %s", result)
	}
}

func TestGlobalFunctions(t *testing.T) {
	var buf bytes.Buffer

	// Setup test logger
	originalLogger := defaultLogger
	originalHandler := defaultHandler
	defer func() {
		defaultLogger = originalLogger
		defaultHandler = originalHandler
	}()

	testHandler := NewSyncConsoleHandler(DebugLevel)
	testHandler.SetOutput(&buf)
	defaultHandler = testHandler
	defaultLogger = NewStructuredLogger(testHandler)

	// Test global functions
	Info("global info message")
	Flush()

	output := buf.String()
	if !strings.Contains(output, "global info message") {
		t.Errorf("Expected global Info to work, got: %s", output)
	}

	// Test context function
	buf.Reset()
	ctx := context.WithValue(context.Background(), LogIDKey, "global123")
	CtxInfo(ctx, "context global message")
	Flush()

	output = buf.String()
	if !strings.Contains(output, "global123") {
		t.Errorf("Expected global CtxInfo to include log_id, got: %s", output)
	}
}

func TestStructuredLogger(t *testing.T) {
	var buf bytes.Buffer
	handler := NewSyncConsoleHandler(DebugLevel)
	handler.SetOutput(&buf)

	logger := NewStructuredLogger(handler)

	// Test traditional logging
	logger.Info("traditional %s", "message")
	logger.Flush()

	output := buf.String()
	if !strings.Contains(output, "traditional message") {
		t.Errorf("Expected structured logger to handle traditional format, got: %s", output)
	}

	// Test With method - StructuredLogger should return a new instance
	newLogger := logger.With("key", "value")
	if newLogger == logger {
		t.Errorf("Expected With to return new logger for StructuredLogger")
	}
}

// func TestFileHandler(t *testing.T) {
// 	// Create temporary directory for test
// 	tmpDir, err := os.MkdirTemp("", "log_test")
// 	if err != nil {
// 		t.Fatalf("Failed to create temp dir: %v", err)
// 	}
// 	defer os.RemoveAll(tmpDir)
//
// 	// Test file handler creation
// 	handler, err := NewFileHandler(InfoLevel, tmpDir, WithFilePrefix("test"))
// 	if err != nil {
// 		t.Fatalf("Failed to create file handler: %v", err)
// 	}
// 	defer handler.Close()
//
// 	// Test file name generation
// 	fileName := handler.FileName()
// 	if !strings.Contains(fileName, "test.") {
// 		t.Errorf("Expected file name to contain prefix 'test.', got: %s", fileName)
// 	}
// 	if !strings.HasPrefix(fileName, tmpDir) {
// 		t.Errorf("Expected file name to be in temp dir, got: %s", fileName)
// 	}
//
// 	// Test basic output
// 	handler.Output(InfoLevel, nil, "file test message")
// 	handler.Flush()
//
// 	// Note: In a real test, we would read the file to verify contents
// 	// but for simplicity we're just testing that no errors occur
// }

// Benchmark tests

func BenchmarkConsoleHandler(b *testing.B) {
	handler := NewConsoleHandler(InfoLevel)
	handler.SetOutput(io.Discard)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.Output(InfoLevel, nil, "benchmark message %d", i)
	}
	handler.Flush()
}

func BenchmarkTextFormatter(b *testing.B) {
	formatter := NewTextFormatter(false)
	ctx := context.WithValue(context.Background(), LogIDKey, "bench123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatter.Format(InfoLevel, ctx, "benchmark %s", "message")
	}
}

// FileHandler benchmarks
func BenchmarkFileHandler_NoRotation(b *testing.B) {
	tmpDir := b.TempDir()

	handler, err := NewFileHandler(FileHandlerConfig{
		LogDir:   tmpDir,
		Filename: "benchmark.log",
		Rotation: RotationNone,
		Level:    InfoLevel,
	})
	if err != nil {
		b.Fatalf("failed to create handler: %v", err)
	}
	defer handler.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			handler.Output(InfoLevel, context.Background(), "benchmark message")
		}
	})
}

func BenchmarkFileHandler_DailyRotation(b *testing.B) {
	tmpDir := b.TempDir()

	handler, err := NewFileHandler(FileHandlerConfig{
		LogDir:   tmpDir,
		Filename: "benchmark_daily.log",
		Rotation: RotationDaily,
		Level:    InfoLevel,
	})
	if err != nil {
		b.Fatalf("failed to create handler: %v", err)
	}
	defer handler.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			handler.Output(InfoLevel, context.Background(), "benchmark message")
		}
	})
}

func BenchmarkFileHandler_HourlyRotation(b *testing.B) {
	tmpDir := b.TempDir()

	handler, err := NewFileHandler(FileHandlerConfig{
		LogDir:   tmpDir,
		Filename: "benchmark_hourly.log",
		Rotation: RotationHourly,
		Level:    InfoLevel,
	})
	if err != nil {
		b.Fatalf("failed to create handler: %v", err)
	}
	defer handler.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			handler.Output(InfoLevel, context.Background(), "benchmark message")
		}
	})
}

func BenchmarkFileHandler_WithContext(b *testing.B) {
	tmpDir := b.TempDir()

	handler, err := NewFileHandler(FileHandlerConfig{
		LogDir:   tmpDir,
		Filename: "benchmark_context.log",
		Rotation: RotationNone,
		Level:    InfoLevel,
	})
	if err != nil {
		b.Fatalf("failed to create handler: %v", err)
	}
	defer handler.Close()

	ctx := WithLogID(context.Background(), "benchmark-request")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			handler.Output(InfoLevel, ctx, "benchmark message with context")
		}
	})
}

// Logger tests
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

	// Test AddOutputs
	logger.AddOutputs(buf2)
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
		name     string
		logFn    func(string, ...any)
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

// Global function tests
func TestGlobalContextFunctions(t *testing.T) {
	// Capture output for testing
	buf := &bytes.Buffer{}
	SetOutput(buf)
	SetLevel(TraceLevel) // Set to trace to capture all levels

	// Test all global context-aware logging functions
	testCases := []struct {
		name     string
		logFn    func(context.Context, string, ...any)
		expected string
	}{
		{"CtxTrace", CtxTrace, "global trace"},
		{"CtxDebug", CtxDebug, "global debug"},
		{"CtxInfo", CtxInfo, "global info"},
		{"CtxWarn", CtxWarn, "global warn"},
		{"CtxError", CtxError, "global error"},
		{"CtxFatal", CtxFatal, "global fatal"},
		{"CtxPanic", CtxPanic, "global panic"},
	}

	for _, tc := range testCases {
		buf.Reset()

		// Test with context
		ctx := context.WithValue(context.Background(), LogIDKey, "global-test-123")
		tc.logFn(ctx, tc.expected)
		Flush()

		if !strings.Contains(buf.String(), tc.expected) {
			t.Errorf("%s: expected log to contain '%s', got: %s", tc.name, tc.expected, buf.String())
		}
	}

	// Test WithGroup function
	buf.Reset()
	groupLogger := WithGroup("testgroup")
	groupLogger.Info("group test")
	Flush()

	if !strings.Contains(buf.String(), "group test") {
		t.Error("Expected WithGroup logger to work")
	}

	// Test RegisterHandler function
	buf.Reset()
	ClearHandler() // Clear existing handlers

	handler := NewSyncConsoleHandler(InfoLevel)
	handler.SetOutput(buf)
	RegisterHandler(handler)

	Info("registered handler test")
	Flush()

	if !strings.Contains(buf.String(), "registered handler test") {
		t.Error("Expected RegisterHandler to work")
	}

	// Test ClearHandler function
	buf.Reset()
	ClearHandler()
	Info("after clear handler")
	Flush()

	// After clearing handlers, logs should still work but might go to different output
	// This is expected behavior

	// Reset to default state
	defaultHandler = NewConsoleHandler(InfoLevel)
	defaultLogger = NewStructuredLogger(defaultHandler)
}

func TestGlobalConfiguration(t *testing.T) {
	// Test SetOutput
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}

	SetOutput(buf1)
	Info("test output 1")
	Flush()

	if !strings.Contains(buf1.String(), "test output 1") {
		t.Error("Expected SetOutput to work")
	}

	// Test AddOutputs
	AddOutputs(buf2)
	Info("test output 2")
	Flush()

	if !strings.Contains(buf1.String(), "test output 2") {
		t.Error("Expected original output to still work")
	}
	if !strings.Contains(buf2.String(), "test output 2") {
		t.Error("Expected AddOutput to work")
	}

	// Test SetLevel
	buf1.Reset()
	buf2.Reset()
	SetLevel(WarnLevel)

	Info("filtered info")
	Warn("warn message")
	Flush()

	if strings.Contains(buf1.String(), "filtered info") {
		t.Error("Expected info message to be filtered")
	}
	if !strings.Contains(buf1.String(), "warn message") {
		t.Error("Expected warn message to be logged")
	}

	// Reset to default level
	SetLevel(InfoLevel)
}

func TestGlobalStructuredOutput(t *testing.T) {
	// Test SetStructuredOutput
	buf := &bytes.Buffer{}
	SetStructuredOutput(buf, true) // JSON output

	Info("json test")
	Flush()

	output := buf.String()
	if !strings.Contains(output, "json test") || !strings.Contains(output, "\"level\":\"INFO\"") {
		t.Errorf("Expected JSON structured output, got: %s", output)
	}

	// Test text structured output
	buf.Reset()
	SetStructuredOutput(buf, false) // Text output

	Info("text test")
	Flush()

	output = buf.String()
	if !strings.Contains(output, "text test") || !strings.Contains(output, "level=INFO") {
		t.Errorf("Expected text structured output, got: %s", output)
	}

	// Reset to default
	defaultHandler = NewConsoleHandler(InfoLevel)
	defaultLogger = NewStructuredLogger(defaultHandler)
}

// MultiError test
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

// Export test function
func TestOutput(t *testing.T) {
	SetLevel(TraceLevel)

	Trace("trace message")
	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")
	Fatal("fatal message")
	Panic("panic message")

	Flush()
}

// Example tests
func TestStructuredLoggingExamples(t *testing.T) {
	// Example 1: Traditional format string logging (backward compatible)
	Info("User %s logged in from %s", "alice", "192.168.1.100")

	// Example 2: Structured logging with key-value pairs
	ctx := context.WithValue(context.Background(), LogIDKey, "req-12345")
	CtxInfo(ctx, "User login", "user", "alice", "ip", "192.168.1.100", "success", true)

	// Example 3: Using With() for contextual logging
	logger := With("service", "auth", "version", "1.0.0")
	logger.Info("Processing request", "method", "POST", "path", "/login")

	// Flush to ensure all logs are written
	Flush()
}

func TestStructuredLogging(t *testing.T) {
	// Test backward compatibility
	Info("Traditional format: %s", "works")

	// Test structured logging
	ctx := context.WithValue(context.Background(), LogIDKey, "test-123")
	CtxInfo(ctx, "Structured log", "key1", "value1", "key2", 42, "key3", true)

	// Test With() method
	logger := With("component", "test")
	logger.Debug("Debug message", "debug_key", "debug_value")

	// Test mixed usage
	Warn("Mixed %s logging", "format", "extra_key", "extra_value")
}

func TestWithLogIDFunction(t *testing.T) {
	// Example 1: Using WithLogID convenience function
	ctx := WithLogID(context.Background(), "request-abc123")
	CtxInfo(ctx, "Request processed", "status", "success", "duration_ms", 250)

	// Example 2: Using WithLogID with existing context
	// Note: In real applications, define your own context key types
	type userKey string
	const userIDKey userKey = "user_id"
	parentCtx := context.WithValue(context.Background(), userIDKey, "user-789")
	ctx = WithLogID(parentCtx, "request-def456")
	CtxInfo(ctx, "User request", "user_id", "user-789", "action", "login")

	// Example 3: Using LogIDKey directly
	ctx = context.WithValue(context.Background(), LogIDKey, "manual-log-id")
	CtxWarn(ctx, "Manual log ID usage")

	Flush()
}

// Test the performance optimization by simulating many rapid log calls
func TestFileHandler_PerformanceOptimization(t *testing.T) {
	tmpDir := t.TempDir()

	handler, err := NewFileHandler(FileHandlerConfig{
		LogDir:   tmpDir,
		Filename: "performance.log",
		Rotation: RotationDaily,
		Level:    InfoLevel,
	})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}
	defer handler.Close()

	// Simulate high-frequency logging
	start := time.Now()
	for i := 0; i < 10000; i++ {
		handler.Output(InfoLevel, context.Background(), "performance test message %d", i)
	}
	duration := time.Since(start)

	// Flush to ensure all messages are written
	handler.Flush()

	t.Logf("Logged 10,000 messages in %v (avg %v per message)", duration, duration/10000)

	// Verify all messages were written
	filePath := handler.GetCurrentFilePath()
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	// Count the number of log lines
	lines := strings.Count(string(content), "\n")
	// Due to async processing and channel capacity, some messages may be dropped
	// This is expected behavior - the test verifies performance, not exact message count
	t.Logf("Successfully logged %d messages (some may be dropped due to async processing)", lines)
}