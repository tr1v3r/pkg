// Package log provides structured logging with multiple output targets,
// async writing, and bidirectional log/slog compatibility.
//
// # Quick Start
//
// One-time setup at program start:
//
//	log.Setup(
//	    log.Console(),                                // colored text → stdout
//	    log.File("app.log"),                          // text → file (async)
//	    log.RotateFile("./logs", "app", log.Daily),   // daily rotation
//	)
//
// Then use package-level functions anywhere:
//
//	log.Info("request received", "method", "GET", "path", "/api")
//	log.Errorf("connection failed: %v", err)
//
// # Structured vs Printf
//
// Two logging styles are supported:
//
//	// Structured — alternating key-value pairs (recommended)
//	log.Info("order created", "order_id", "ORD-001", "amount", 299.99)
//	// Output: 2024-01-01T12:00:00Z [INFO] order created order_id=ORD-001 amount=299.99
//
//	// Printf — format string
//	log.Infof("server started on :%d", 8080)
//	// Output: 2024-01-01T12:00:00Z [INFO] server started on :8080
//
// Each level (Trace/Debug/Info/Warn/Error/Fatal) has four forms:
//
//	Info(msg, args...)     // structured
//	Infof(format, args...) // printf
//	CtxInfo(ctx, msg, args...)   // structured + context (extracts logID)
//	CtxInfof(ctx, format, args...) // printf + context
//
// # Context and logID
//
// Use WithLogID to inject a request-scoped ID into context. All Ctx methods
// automatically extract and include it in the output:
//
//	func middleware(ctx context.Context) context.Context {
//	    return log.WithLogID(ctx, log.NewLogID())
//	}
//
//	func handler(ctx context.Context) {
//	    log.CtxInfo(ctx, "processing", "step", "validate")
//	    // Output: ... [INFO] [a1b2c3d4-...] processing step=validate
//	}
//
// # Sub-Loggers
//
// With returns a child Logger with preset fields that appear in every log entry:
//
//	reqLog := log.With("service", "order", "version", "2.0")
//	reqLog.Info("created", "order_id", "001")
//	// Output: ... [INFO] created service=order version=2.0 order_id=001
//
// # Typed Fields
//
// For type safety, use typed Field constructors (interchangeable with key-value):
//
//	log.Info("result",
//	    log.Int("count", 42),
//	    log.Err(err),
//	    log.Duration("elapsed", 150*time.Millisecond),
//	)
//
// Both styles can be mixed in the same call:
//
//	log.Info("mixed", "key", "val", log.Int("count", 3))
//
// # Sink Types
//
// Console — colored text to stdout:
//
//	log.Console()                      // default: InfoLevel, sync
//	log.Console(os.Stderr, log.WithLevel(log.DebugLevel))
//
// File — plain text to a single file:
//
//	log.File("app.log")                // default: InfoLevel, async (buf=1024)
//
// RotateFile — time-based rotation:
//
//	log.RotateFile("./logs", "app", log.Hourly)   // app_2024-01-01_15.log
//	log.RotateFile("./logs", "app", log.Daily)    // app_2024-01-01.log
//	log.RotateFile("./logs", "app", log.Weekly)   // app_2024-W01.log
//	log.RotateFile("./logs", "app", log.Monthly)  // app_2024-01.log
//
// SizeRotateFile — size-based rotation:
//
//	log.SizeRotateFile("./logs", "app", 100*1024*1024) // rotate at 100MB
//	// Files: app_001.log, app_002.log, ...
//
// # Async Writing
//
// File, RotateFile, and SizeRotateFile use async buffered writing by default
// (buffer size 1024). Use WithSync() to override back to synchronous:
//
//	sink, _ := log.File("app.log", log.WithSync())
//
// Call Sync() or Close() to flush buffered data before exit:
//
//	defer log.Sync()
//
// # io.Writer Compatibility
//
// Sink implements io.Writer, allowing it to replace gin.DefaultWriter or any
// io.Writer destination:
//
//	sink, _ := log.File("gin.log")
//	gin.DefaultWriter = sink
//	gin.DefaultErrorWriter = sink
//
// # slog Interoperability
//
// Two-way compatibility with log/slog:
//
//	// Use our Sink as slog.Handler (slog → our pipeline):
//	sink, _ := log.RotateFile("./logs", "app", log.Daily)
//	slog.SetDefault(slog.New(log.AsSlogHandler(sink)))
//
//	// Use slog.Handler as our Sink (our logs → slog ecosystem):
//	sentryHandler := sentryslog.NewSentryHandler(...)
//	log.Setup(log.SlogHandler(sentryHandler))
//
// # Level
//
// From lowest to highest: Trace < Debug < Info < Warn < Error < Fatal.
// Fatal logs the message then calls os.Exit(1).
// Change level at runtime:
//
//	log.SetLevel(log.DebugLevel)   // package-level
//	sink.SetLevel(log.WarnLevel)   // per-sink
package log
