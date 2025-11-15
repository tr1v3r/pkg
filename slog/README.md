# Slog Package - slog Interface Compatibility for log Package

This package provides slog interface compatibility for the existing `log` package, allowing you to use Go's standard `slog` interface with the existing logging infrastructure.

## Features

- **Full slog compatibility**: Use standard `slog.Logger` interface with existing log handlers
- **Seamless integration**: No changes required to existing logging code
- **Level mapping**: Automatic conversion between log package levels and slog levels
- **Extended interfaces**: Both original and slog-compatible interfaces available
- **Multiple handlers**: Support for wrapping multiple log handlers

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log/slog"
    "github.com/tr1v3r/pkg/log"
    "github.com/tr1v3r/pkg/slog"
)

func main() {
    // Create logger using existing log package
    logger := log.NewLogger(log.NewConsoleHandler(log.InfoLevel))

    // Use it as before - no changes needed
    logger.Info("Traditional logging")

    // Convert to slog logger for slog compatibility
    slogger := slog.AsSlogLogger(logger)
    slogger.Info("Slog compatibility", "version", "1.0.0")

    // Use extended logger for both interfaces
    extLogger := slog.AsExtendedLogger(logger)

    // Original interface still works
    extLogger.Info("Original interface")

    // New slog-compatible methods
    extLogger.InfoCtx(context.Background(), "Context logging", "key", "value")
}
```

### Handler Wrapping

```go
// Wrap existing handlers for slog usage
consoleHandler := log.NewConsoleHandler(log.InfoLevel)
fileHandler, _ := log.NewFileHandler(log.FileHandlerConfig{
    Filename: "app.log",
    Level:    log.DebugLevel,
})

// Wrap individual handlers
consoleSlogger := slog.New(WrapHandler(consoleHandler))
fileSlogger := slog.New(WrapHandler(fileHandler))

// Wrap multiple handlers
multiSlogger := slog.New(WrapHandlers(consoleHandler, fileHandler))
```

### Level Mapping

The package provides automatic conversion between log package levels and slog levels:

```go
log.DebugLevel → slog.LevelDebug
log.InfoLevel → slog.LevelInfo
log.WarnLevel → slog.LevelWarn
log.ErrorLevel → slog.LevelError
log.FatalLevel → slog.LevelError + 4
log.PanicLevel → slog.LevelError + 8
log.TraceLevel → slog.LevelDebug - 4
```

## Migration Strategy

### Gradual Migration

1. **Keep existing code unchanged**:
   ```go
   logger := log.NewLogger(log.NewConsoleHandler(log.InfoLevel))
   logger.Info("Existing code works")
   ```

2. **Add slog functionality where needed**:
   ```go
   slogger := slog.AsSlogLogger(logger)
   slogger.Info("New slog-style logging", "feature", "analytics")
   ```

3. **Use extended logger for mixed usage**:
   ```go
   extLogger := slog.AsExtendedLogger(logger)
   extLogger.Info("Both interfaces available")
   ```

### Performance Considerations

- Use original interface for high-frequency logging
- Use slog interface when structured logging is needed
- Both interfaces share the same underlying handlers

```go
// High performance: use original interface
logger.Debug("Processing item %d", i)

// Structured logging: use slog interface
slogger.Info("Request processed",
    "request_id", "req-123",
    "latency_ms", 45,
    "status_code", 200,
)
```

## API Reference

### Main Functions

- `slog.AsSlogLogger(logger log.Logger) *slog.Logger` - Convert to slog.Logger
- `slog.AsExtendedLogger(logger log.Logger) ExtendedLogger` - Get extended interface
- `slog.WrapHandler(handler log.Handler) slog.Handler` - Wrap single handler
- `slog.WrapHandlers(handlers ...log.Handler) slog.Handler` - Wrap multiple handlers

### Level Conversion

- `slog.ToSlogLevel(level log.Level) slog.Level` - Convert log level to slog
- `slog.FromSlogLevel(level slog.Level) log.Level` - Convert slog level to log
- `slog.IsLevelEnabled(currentLogLevel log.Level, slogLevel slog.Level) bool` - Check if level is enabled

### ExtendedLogger Interface

Extends `log.Logger` with slog-compatible methods:

- `Log(ctx context.Context, level slog.Level, msg string, args ...any)`
- `LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr)`
- `DebugCtx(ctx context.Context, msg string, args ...any)`
- `InfoCtx(ctx context.Context, msg string, args ...any)`
- `WarnCtx(ctx context.Context, msg string, args ...any)`
- `ErrorCtx(ctx context.Context, msg string, args ...any)`

## Examples

See `example_test.go` for comprehensive usage examples including:

- Basic usage patterns
- Context-aware logging
- Multiple handler scenarios
- Performance considerations
- Migration strategies

## Compatibility

- Maintains full backward compatibility with existing `log` package
- Compatible with Go 1.21+ slog interface
- Works with all existing log package handlers
- No changes required to existing code