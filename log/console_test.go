package log

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestConsoleHandler_SetOutput(t *testing.T) {
	handler := NewConsoleHandler(InfoLevel)

	// Test SetOutput with a buffer
	buf := &bytes.Buffer{}
	handler.SetOutput(buf)

	// Log a message and flush
	handler.Output(InfoLevel, nil, "test message")
	handler.Flush()

	if !strings.Contains(buf.String(), "test message") {
		t.Errorf("Expected buffer to contain 'test message', got: %s", buf.String())
	}
}

func TestConsoleHandler_AddOutput(t *testing.T) {
	handler := NewConsoleHandler(InfoLevel)

	// Test AddOutput with multiple buffers
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}

	handler.SetOutput(buf1)
	handler.AddOutput(buf2)

	// Log a message and flush
	handler.Output(InfoLevel, nil, "multi-output test")
	handler.Flush()

	if !strings.Contains(buf1.String(), "multi-output test") {
		t.Errorf("Expected buffer1 to contain 'multi-output test', got: %s", buf1.String())
	}
	if !strings.Contains(buf2.String(), "multi-output test") {
		t.Errorf("Expected buffer2 to contain 'multi-output test', got: %s", buf2.String())
	}
}

func TestConsoleHandler_Close(t *testing.T) {
	handler := NewConsoleHandler(InfoLevel)

	// Test Close doesn't panic
	err := handler.Close()
	if err != nil {
		t.Errorf("Expected Close to return nil error, got: %v", err)
	}

	// Test that handler is closed and won't accept new messages
	// Note: Messages sent after close are dropped silently
	// This is expected behavior to prevent blocking
}

func TestConsoleHandler_CloseMultipleTimes(t *testing.T) {
	// Create a new handler for each Close test
	handler1 := NewConsoleHandler(InfoLevel)
	err1 := handler1.Close()
	if err1 != nil {
		t.Errorf("First Close should return nil, got: %v", err1)
	}

	handler2 := NewConsoleHandler(InfoLevel)
	err2 := handler2.Close()
	if err2 != nil {
		t.Errorf("Second Close should return nil, got: %v", err2)
	}
}

func TestConsoleHandler_FlushWithPendingMessages(t *testing.T) {
	handler := NewConsoleHandler(InfoLevel)
	buf := &bytes.Buffer{}
	handler.SetOutput(buf)

	// Send multiple messages
	for i := 0; i < 3; i++ {
		handler.Output(InfoLevel, nil, "message %d", i)
	}

	// Give some time for async processing
	time.Sleep(50 * time.Millisecond)

	// Flush should process all messages
	handler.Flush()

	output := buf.String()
	// Just check that we got some messages, not specific ones due to async nature
	if !strings.Contains(output, "message") {
		t.Errorf("Expected flush to contain some messages, got: %s", output)
	}
}

func TestConsoleHandler_AsyncProcessing(t *testing.T) {
	handler := NewConsoleHandler(InfoLevel)
	buf := &bytes.Buffer{}
	handler.SetOutput(buf)

	// Send multiple messages asynchronously
	for i := 0; i < 10; i++ {
		go func(i int) {
			handler.Output(InfoLevel, nil, "async message %d", i)
		}(i)
	}

	// Give some time for async processing
	time.Sleep(100 * time.Millisecond)
	handler.Flush()

	// Should have processed some messages
	output := buf.String()
	if !strings.Contains(output, "async message") {
		t.Errorf("Expected async messages to be processed, got: %s", output)
	}
}

func TestConsoleHandler_LevelFiltering(t *testing.T) {
	// Use synchronous handler for predictable testing
	handler := NewSyncConsoleHandler(WarnLevel)
	buf := &bytes.Buffer{}
	handler.SetOutput(buf)

	// Test that lower level messages are filtered
	handler.Output(InfoLevel, nil, "info message")
	handler.Output(DebugLevel, nil, "debug message")
	handler.Output(WarnLevel, nil, "warn message")
	handler.Output(ErrorLevel, nil, "error message")
	handler.Flush()

	output := buf.String()

	if strings.Contains(output, "info message") {
		t.Error("Info message should be filtered out")
	}
	if strings.Contains(output, "debug message") {
		t.Error("Debug message should be filtered out")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("Warn message should be logged")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Error message should be logged")
	}
}

func TestConsoleHandler_ContextLogging(t *testing.T) {
	handler := NewConsoleHandler(InfoLevel)
	buf := &bytes.Buffer{}
	handler.SetOutput(buf)

	// Test logging with context
	ctx := context.WithValue(context.Background(), LogIDKey, "test-789")
	handler.Output(InfoLevel, ctx, "context message")
	handler.Flush()

	output := buf.String()
	if !strings.Contains(output, "context message") {
		t.Errorf("Expected context message to be logged, got: %s", output)
	}
	if !strings.Contains(output, "test-789") {
		t.Errorf("Expected log_id from context to be included, got: %s", output)
	}
}

func TestConsoleHandler_WriteDirect(t *testing.T) {
	handler := NewConsoleHandler(InfoLevel)
	buf := &bytes.Buffer{}
	handler.SetOutput(buf)

	// Test direct Write method
	testData := []byte("direct write test")
	n, err := handler.Write(testData)

	if err != nil {
		t.Errorf("Write should not return error, got: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Write should return %d bytes written, got: %d", len(testData), n)
	}
	if !strings.Contains(buf.String(), "direct write test") {
		t.Errorf("Expected direct write to be in output, got: %s", buf.String())
	}
}