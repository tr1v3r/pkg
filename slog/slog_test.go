package slog

import (
	"bytes"
	"context"
	stdslog "log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/tr1v3r/pkg/log"
)

func TestLevelConversion(t *testing.T) {
	tests := []struct {
		name         string
		logLevel     log.Level
		expectedSlog stdslog.Level
	}{
		{"Trace to Slog", log.TraceLevel, stdslog.LevelDebug - 4},
		{"Debug to Slog", log.DebugLevel, stdslog.LevelDebug},
		{"Info to Slog", log.InfoLevel, stdslog.LevelInfo},
		{"Warn to Slog", log.WarnLevel, stdslog.LevelWarn},
		{"Error to Slog", log.ErrorLevel, stdslog.LevelError},
		{"Fatal to Slog", log.FatalLevel, stdslog.LevelError + 4},
		{"Panic to Slog", log.PanicLevel, stdslog.LevelError + 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slogLevel := ToSlogLevel(tt.logLevel)
			assert.Equal(t, tt.expectedSlog, slogLevel)

			// Test round-trip conversion
			convertedBack := FromSlogLevel(slogLevel)
			assert.Equal(t, tt.logLevel, convertedBack)
		})
	}
}

func TestSlogToLogLevelConversion(t *testing.T) {
	tests := []struct {
		name        string
		slogLevel   stdslog.Level
		expectedLog log.Level
	}{
		{"Low custom levels", stdslog.LevelDebug - 8, log.TraceLevel},
		{"Trace range", stdslog.LevelDebug - 2, log.TraceLevel},
		{"Debug range", stdslog.LevelDebug, log.DebugLevel},
		{"Info range", stdslog.LevelInfo, log.InfoLevel},
		{"Warn range", stdslog.LevelWarn, log.WarnLevel},
		{"Error range", stdslog.LevelError, log.ErrorLevel},
		{"Fatal range", stdslog.LevelError + 5, log.FatalLevel},
		{"Panic range", stdslog.LevelError + 10, log.PanicLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logLevel := FromSlogLevel(tt.slogLevel)
			assert.Equal(t, tt.expectedLog, logLevel)
		})
	}
}

func TestSlogHandlerAdapter(t *testing.T) {
	handler := log.NewConsoleHandler(log.InfoLevel)
	adapter := NewSlogHandlerAdapter(handler)

	// Test Enabled
	ctx := context.Background()
	assert.True(t, adapter.Enabled(ctx, stdslog.LevelInfo))
	assert.False(t, adapter.Enabled(ctx, stdslog.LevelDebug))

	// Test Handle - just test that it doesn't error
	record := stdslog.NewRecord(time.Now(), stdslog.LevelInfo, "Test message", 0)
	record.Add("key", "value")
	record.Add("number", 42)

	err := adapter.Handle(ctx, record)
	assert.NoError(t, err)

	// Test WithAttrs
	attrs := []stdslog.Attr{
		stdslog.String("attr1", "value1"),
		stdslog.Int("attr2", 123),
	}
	newAdapter := adapter.WithAttrs(attrs)
	assert.NotNil(t, newAdapter)

	// Test WithGroup
	groupAdapter := adapter.WithGroup("test")
	assert.NotNil(t, groupAdapter)
}

func TestLoggerWrapper(t *testing.T) {
	originalLogger := log.NewLogger(log.NewConsoleHandler(log.InfoLevel))
	wrapper := NewLoggerWrapper(originalLogger)

	// Test slog compatibility methods - just ensure they don't panic
	ctx := context.Background()
	assert.NotPanics(t, func() {
		wrapper.InfoCtx(ctx, "Test message", "key", "value")
	})

	assert.NotPanics(t, func() {
		wrapper.DebugCtx(ctx, "Debug message", "key", "value")
	})

	// Test LogAttrs
	assert.NotPanics(t, func() {
		wrapper.LogAttrs(ctx, stdslog.LevelInfo, "Attrs message",
			stdslog.String("string", "value"),
			stdslog.Int("number", 42),
		)
	})
}

func TestAsSlogLogger(t *testing.T) {
	originalLogger := log.NewLogger(log.NewConsoleHandler(log.InfoLevel))
	slogger := AsSlogLogger(originalLogger)

	// Test that it doesn't panic
	assert.NotPanics(t, func() {
		slogger.Info("Slog message", "key", "value")
	})

	assert.NotPanics(t, func() {
		slogger.Debug("Debug message", "key", "value")
	})

	// Test that it returns a valid slog.Logger
	assert.NotNil(t, slogger)
}

func TestMultiHandlerAdapter(t *testing.T) {
	handler1 := log.NewConsoleHandler(log.InfoLevel)
	handler2 := log.NewConsoleHandler(log.DebugLevel)

	multiAdapter := &MultiHandlerAdapter{
		handlers: []log.Handler{handler1, handler2},
	}

	ctx := context.Background()
	record := stdslog.NewRecord(time.Now(), stdslog.LevelInfo, "Multi test", 0)
	record.Add("key", "value")

	err := multiAdapter.Handle(ctx, record)
	assert.NoError(t, err)

	// Test WithAttrs
	attrs := []stdslog.Attr{stdslog.String("group", "test")}
	newAdapter := multiAdapter.WithAttrs(attrs)
	assert.NotNil(t, newAdapter)

	// Test WithGroup
	groupAdapter := multiAdapter.WithGroup("test")
	assert.NotNil(t, groupAdapter)
}

func TestWrapHandler(t *testing.T) {
	handler := log.NewConsoleHandler(log.InfoLevel)
	wrapped := WrapHandler(handler)

	assert.NotNil(t, wrapped)

	// Test that it implements slog.Handler interface
	assert.Implements(t, (*stdslog.Handler)(nil), wrapped)
}

func TestWrapHandlers(t *testing.T) {
	handler1 := log.NewConsoleHandler(log.InfoLevel)
	handler2 := log.NewConsoleHandler(log.DebugLevel)

	// Single handler
	single := WrapHandlers(handler1)
	assert.IsType(t, &SlogHandlerAdapter{}, single)

	// Multiple handlers
	multi := WrapHandlers(handler1, handler2)
	assert.IsType(t, &MultiHandlerAdapter{}, multi)
}

func TestAsExtendedLogger(t *testing.T) {
	originalLogger := log.NewLogger(log.NewConsoleHandler(log.InfoLevel))
	extLogger := AsExtendedLogger(originalLogger)

	// Test that it implements both interfaces
	assert.Implements(t, (*log.Logger)(nil), extLogger)
	assert.Implements(t, (*ExtendedLogger)(nil), extLogger)

	// Test original interface methods
	extLogger.Info("Original interface test")

	// Test extended interface methods
	ctx := context.Background()
	extLogger.InfoCtx(ctx, "Extended interface test", "key", "value")
}

func TestContextIntegration(t *testing.T) {
	originalLogger := log.NewLogger(log.NewConsoleHandler(log.InfoLevel))
	extLogger := AsExtendedLogger(originalLogger)

	// Create context with log ID
	ctx := log.WithLogID(context.Background(), "test-123")

	// Log with context - just ensure it doesn't panic
	assert.NotPanics(t, func() {
		extLogger.InfoCtx(ctx, "Context test")
	})
}

func BenchmarkLogLevelConversion(b *testing.B) {
	levels := []log.Level{
		log.TraceLevel, log.DebugLevel, log.InfoLevel,
		log.WarnLevel, log.ErrorLevel, log.FatalLevel, log.PanicLevel,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		level := levels[i%len(levels)]
		_ = ToSlogLevel(level)
	}
}

func BenchmarkSlogHandlerAdapter_Handle(b *testing.B) {
	var buf bytes.Buffer
	handler := log.NewConsoleHandler(log.InfoLevel)
	handler.SetOutput(&buf)
	adapter := NewSlogHandlerAdapter(handler)

	ctx := context.Background()
	record := stdslog.NewRecord(time.Now(), stdslog.LevelInfo, "Benchmark message", 0)
	record.Add("iteration", 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Update record with current iteration
		record = stdslog.NewRecord(time.Now(), stdslog.LevelInfo, "Benchmark message", 0)
		record.Add("iteration", i)
		_ = adapter.Handle(ctx, record)
	}
}

func BenchmarkAsSlogLogger(b *testing.B) {
	logger := log.NewLogger(log.NewConsoleHandler(log.InfoLevel))
	slogger := AsSlogLogger(logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slogger.Info("Benchmark message", "iteration", i)
	}
}
