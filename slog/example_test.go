package slog

import (
	"context"
	stdslog "log/slog"
	"testing"
	"time"

	"github.com/tr1v3r/pkg/log"
)

// contextKey is used to avoid context key collisions
type contextKey string

const (
	userIDKey contextKey = "user_id"
)

// ExampleLogger demonstrates how to use the slog package with existing log package
func ExampleLogger() {
	// Create a logger using existing log package
	fileHandler, err := log.NewFileHandler(log.FileHandlerConfig{
		Filename: "app.log",
		Level:    log.DebugLevel,
		Rotation: log.RotationHourly,
	})
	if err != nil {
		panic(err)
	}
	logger := log.NewLogger(
		log.NewConsoleHandler(log.InfoLevel),
		fileHandler,
	)

	// 1. Use it as before - no changes needed
	logger.Info("Starting application")

	// 2. Convert to slog.Logger for slog compatibility
	slogger := AsSlogLogger(logger)
	slogger.Info("Starting application", "version", "1.0.0", "port", 8080)

	// 3. Use extended logger for both interfaces
	extLogger := AsExtendedLogger(logger)

	// Original interface still works
	extLogger.Info("Application started")

	// New slog-compatible methods
	extLogger.InfoCtx(context.Background(), "Request processed",
		"method", "GET",
		"path", "/api/users",
		"duration", 150*time.Millisecond)

	// Use LogAttrs for structured logging
	extLogger.LogAttrs(context.Background(),
		stdslog.LevelInfo,
		"User action",
		stdslog.String("user_id", "12345"),
		stdslog.String("action", "login"),
		stdslog.Time("timestamp", time.Now()),
	)
}

// ExampleContext demonstrates context-aware logging
func ExampleContext() {
	logger := log.NewLogger(log.NewConsoleHandler(log.DebugLevel))
	extLogger := AsExtendedLogger(logger)

	// Create context with log ID
	ctx := log.WithLogID(context.Background(), "request-123")

	// Add additional context values
	ctx = context.WithValue(ctx, userIDKey, "user-456")

	// Log with context - log ID will be included
	extLogger.InfoCtx(ctx, "Processing request")
	extLogger.DebugCtx(ctx, "Database query executed",
		"query", "SELECT * FROM users",
		"rows", 10)

	// Using slog interface with context
	slogger := AsSlogLogger(logger)
	slogger.InfoContext(ctx, "Request completed successfully")
}

// TestMigration demonstrates gradual migration from log package to slog
func TestMigration(t *testing.T) {
	// Step 1: Keep existing code unchanged
	logger := log.NewLogger(log.NewConsoleHandler(log.InfoLevel))
	logger.Info("Existing code works")

	// Step 2: Add slog functionality where needed
	slogger := AsSlogLogger(logger)
	slogger.Info("New slog-style logging", "feature", "analytics")

	// Step 3: Use extended logger for mixed usage
	extLogger := AsExtendedLogger(logger)
	extLogger.Info("Both interfaces available")

	// Step 4: Gradually migrate specific areas to use slog interface
	processRequest(slogger)
}

func processRequest(slogger *stdslog.Logger) {
	// New function using standard slog interface
	slogger.Info("Processing request",
		"handler", "user_profile",
		"method", "GET",
	)
}

// TestMulti demonstrates using multiple handlers with slog
func TestMulti(t *testing.T) {
	consoleHandler := log.NewConsoleHandler(log.InfoLevel)
	fileHandler, _ := log.NewFileHandler(log.FileHandlerConfig{
		Filename: "debug.log",
		Level:    log.DebugLevel,
		Rotation: log.RotationDaily,
	})
	// structuredHandler is not available in log package, skip for now
	// structuredHandler := log.NewStructuredLogHandler(log.WarnLevel)

	// Wrap multiple handlers for slog compatibility
	slogHandler := WrapHandlers(consoleHandler, fileHandler)
	slogger := stdslog.New(slogHandler)

	// Log message will go to all handlers
	slogger.Info("Multi-handler logging test",
		"component", "example",
		"handlers", "console,file",
	)

	slogger.Warn("Warning message",
		"component", "example",
		"severity", "high",
	)
}

// TestWrapping demonstrates individual handler wrapping
func TestWrapping(t *testing.T) {
	// Create individual handlers
	consoleHandler := log.NewConsoleHandler(log.InfoLevel)
	fileHandler, _ := log.NewFileHandler(log.FileHandlerConfig{
		Filename: "app.log",
		Level:    log.DebugLevel,
		Rotation: log.RotationHourly,
	})

	// Wrap individual handlers for different slog loggers
	consoleSlogger := stdslog.New(WrapHandler(consoleHandler))
	fileSlogger := stdslog.New(WrapHandler(fileHandler))

	// Use different loggers for different purposes
	consoleSlogger.Info("Console message", "destination", "console")
	fileSlogger.Debug("File message", "destination", "file")
}

// TestConversion demonstrates level mapping between packages
func TestConversion(t *testing.T) {
	logger := log.NewLogger(log.NewConsoleHandler(log.InfoLevel))

	// Show level conversion
	logLevels := []log.Level{
		log.TraceLevel,
		log.DebugLevel,
		log.InfoLevel,
		log.WarnLevel,
		log.ErrorLevel,
		log.FatalLevel,
		log.PanicLevel,
	}

	for _, level := range logLevels {
		slogLevel := ToSlogLevel(level)
		convertedBack := FromSlogLevel(slogLevel)

		logger.Info("Level conversion test",
			"original", level.String(),
			"slog", slogLevel.String(),
			"converted_back", convertedBack.String(),
		)
	}
}

// TestPerformance demonstrates performance considerations
func TestPerformance(t *testing.T) {
	fileHandler, _ := log.NewFileHandler(log.FileHandlerConfig{
		Filename: "perf.log",
		Level:    log.InfoLevel,
		Rotation: log.RotationHourly,
	})
	logger := log.NewLogger(
		log.NewConsoleHandler(log.InfoLevel),
		fileHandler,
	)

	// Use original interface for high-performance logging
	logger.Info("High performance logging with original interface")

	// Use slog interface when structured logging is needed
	slogger := AsSlogLogger(logger)
	slogger.Info("Structured logging when needed",
		"request_id", "req-123",
		"latency_ms", 45,
		"status_code", 200,
	)

	// Choose interface based on use case
	for i := 0; i < 100; i++ {
		// High frequency: use original interface
		logger.Debug("Processing item %d", i)

		if i%10 == 0 {
			// Periodic structured logging: use slog interface
			slogger.Info("Batch progress", "processed", i, "total", 100)
		}
	}
}
