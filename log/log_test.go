package log

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
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

func TestConsoleHandler(t *testing.T) {
	var buf bytes.Buffer
	handler := NewSyncConsoleHandler(DebugLevel)
	handler.SetOutput(&buf)

	// Test basic output
	handler.Output(InfoLevel, nil, "test message")
	handler.Flush()

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected output to contain 'test message', got: %s", output)
	}
	if !strings.Contains(output, "INFO") {
		t.Errorf("Expected output to contain 'INFO', got: %s", output)
	}

	// Test level filtering
	buf.Reset()
	handler.SetLevel(WarnLevel)
	handler.Output(InfoLevel, nil, "should not appear")
	handler.Output(WarnLevel, nil, "should appear")
	handler.Flush()

	output = buf.String()
	if strings.Contains(output, "should not appear") {
		t.Errorf("Info level message should have been filtered out")
	}
	if !strings.Contains(output, "should appear") {
		t.Errorf("Warn level message should have been included")
	}

	// Test context
	buf.Reset()
	handler.SetLevel(DebugLevel)
	ctx := context.WithValue(context.Background(), "log_id", "test123")
	handler.Output(InfoLevel, ctx, "context message")
	handler.Flush()

	output = buf.String()
	t.Logf("Output with context: %q", output)
	if !strings.Contains(output, "test123") {
		t.Errorf("Expected output to contain log_id 'test123', got: %s", output)
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
	ctx := context.WithValue(context.Background(), "log_id", "ctx123")
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
	ctx := context.WithValue(context.Background(), "log_id", "global123")
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
	ctx := context.WithValue(context.Background(), "log_id", "bench123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatter.Format(InfoLevel, ctx, "benchmark %s", "message")
	}
}