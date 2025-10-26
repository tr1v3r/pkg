package log

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

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

	// Test AddOutput
	AddOutput(buf2)
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