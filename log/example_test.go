package log

import (
	"context"
	"os"
	"testing"
)

// TestStructuredLoggingExamples demonstrates various logging patterns
func TestStructuredLoggingExamples(t *testing.T) {
	// Example 1: Traditional format string logging (backward compatible)
	Info("User %s logged in from %s", "alice", "192.168.1.100")

	// Example 2: Structured logging with key-value pairs
	ctx := context.WithValue(context.Background(), "log_id", "req-12345")
	CtxInfo(ctx, "User login", "user", "alice", "ip", "192.168.1.100", "success", true)

	// Example 3: Using With() for contextual logging
	logger := With("service", "auth", "version", "1.0.0")
	logger.Info("Processing request", "method", "POST", "path", "/login")

	// Example 4: JSON structured output
	SetStructuredOutput(os.Stdout, true)
	CtxInfo(ctx, "User action", "action", "login", "duration_ms", 150, "user_agent", "Mozilla/5.0")

	// Flush to ensure all logs are written
	Flush()
}

func TestStructuredLogging(t *testing.T) {
	// Test backward compatibility
	Info("Traditional format: %s", "works")

	// Test structured logging
	ctx := context.WithValue(context.Background(), "log_id", "test-123")
	CtxInfo(ctx, "Structured log", "key1", "value1", "key2", 42, "key3", true)

	// Test With() method
	logger := With("component", "test")
	logger.Debug("Debug message", "debug_key", "debug_value")

	// Test mixed usage
	Warn("Mixed %s logging", "format", "extra_key", "extra_value")
}