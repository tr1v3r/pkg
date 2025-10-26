package log

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

// TestConsoleHandler tests basic ConsoleHandler functionality
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
}

// TestConsoleHandler_SetOutput tests SetOutput functionality
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

// TestConsoleHandler_AddOutputs tests AddOutputs functionality
func TestConsoleHandler_AddOutputs(t *testing.T) {
	handler := NewConsoleHandler(InfoLevel)

	// Test AddOutputs with multiple buffers
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}

	handler.SetOutput(buf1)
	handler.AddOutputs(buf2)

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

// TestConsoleHandler_Close tests Close functionality
func TestConsoleHandler_Close(t *testing.T) {
	handler := NewConsoleHandler(InfoLevel)

	// Test that Close doesn't panic
	handler.Close()
}


// TestConsoleHandler_FlushWithPendingMessages tests Flush with pending messages
func TestConsoleHandler_FlushWithPendingMessages(t *testing.T) {
	handler := NewConsoleHandler(InfoLevel)

	// Log multiple messages
	for i := 0; i < 100; i++ {
		handler.Output(InfoLevel, nil, "message %d", i)
	}

	// Flush should process all messages
	handler.Flush()

	// Close should work after flush
	handler.Close()
}

// TestConsoleHandler_AsyncProcessing tests async message processing
func TestConsoleHandler_AsyncProcessing(t *testing.T) {
	handler := NewConsoleHandler(InfoLevel)

	// Log many messages asynchronously
	for i := 0; i < 1000; i++ {
		handler.Output(InfoLevel, nil, "async message %d", i)
	}

	// Give some time for async processing
	time.Sleep(100 * time.Millisecond)

	// Flush to ensure all messages are processed
	handler.Flush()

	// Close the handler
	handler.Close()
}

// TestConsoleHandler_LevelFiltering tests level filtering
func TestConsoleHandler_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	handler := NewConsoleHandler(InfoLevel)
	handler.SetOutput(&buf)

	// Log messages at different levels
	handler.Output(DebugLevel, nil, "debug message")   // Should be filtered
	handler.Output(InfoLevel, nil, "info message")     // Should pass
	handler.Output(WarnLevel, nil, "warn message")     // Should pass
	handler.Output(ErrorLevel, nil, "error message")   // Should pass

	handler.Flush()

	output := buf.String()

	// Debug message should be filtered out
	if strings.Contains(output, "debug message") {
		t.Error("Debug message should be filtered out")
	}

	// Info, warn, and error messages should be present
	if !strings.Contains(output, "info message") {
		t.Error("Info message should be present")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("Warn message should be present")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Error message should be present")
	}
}

// TestConsoleHandler_ContextLogging tests logging with context
func TestConsoleHandler_ContextLogging(t *testing.T) {
	var buf bytes.Buffer
	handler := NewConsoleHandler(InfoLevel)
	handler.SetOutput(&buf)

	// Create context with log ID
	ctx := context.WithValue(context.Background(), LogIDKey, "test-123")

	// Log with context
	handler.Output(InfoLevel, ctx, "context message")
	handler.Flush()

	output := buf.String()
	if !strings.Contains(output, "context message") {
		t.Error("Expected output to contain context message")
	}
	if !strings.Contains(output, "test-123") {
		t.Error("Expected output to contain log ID from context")
	}
}

// TestConsoleHandler_WriteDirect tests direct Write method
func TestConsoleHandler_WriteDirect(t *testing.T) {
	var buf bytes.Buffer
	handler := NewConsoleHandler(InfoLevel)
	handler.SetOutput(&buf)

	// Test direct Write method
	data := []byte("direct write test\n")
	n, err := handler.Write(data)

	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	// Verify the data was written
	if !strings.Contains(buf.String(), "direct write test") {
		t.Error("Expected buffer to contain direct write test")
	}
}