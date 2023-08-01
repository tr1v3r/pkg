package log

import (
	"context"
	"io"
	"sync"
)

var defaultHandler = NewStreamHandler(InfoLevel)
var defaultLogger = NewLogger(defaultHandler)

// NewLogger new logger
func NewLogger(handlers ...Handler) Logger { return &logger{handlers: handlers} }

// Logger logger interface
type Logger interface {
	RegisterHandler(...Handler)
	SetLevel(Level)

	Flush()
	Close()

	Trace(format string, v ...any)
	Debug(format string, v ...any)
	Info(format string, v ...any)
	Warn(format string, v ...any)
	Error(format string, v ...any)
	Fatal(format string, v ...any)
	Panic(format string, v ...any)

	CtxTrace(ctx context.Context, format string, v ...any)
	CtxDebug(ctx context.Context, format string, v ...any)
	CtxInfo(ctx context.Context, format string, v ...any)
	CtxWarn(ctx context.Context, format string, v ...any)
	CtxError(ctx context.Context, format string, v ...any)
	CtxFatal(ctx context.Context, format string, v ...any)
	CtxPanic(ctx context.Context, format string, v ...any)
}

type Handler interface {
	Output(level Level, ctx context.Context, format string, v ...any)

	SetLevel(Level)
	RegisterOutput(io.Writer)
	Flush()
	Close()
}

type logger struct {
	mu       sync.RWMutex
	handlers []Handler
}

func (l *logger) SetLevel(level Level) {
	l.mu.RLock()
	defer l.mu.RLock()
	for _, handler := range l.handlers {
		handler.SetLevel(level)
	}
}

func (l *logger) RegisterHandler(handlers ...Handler) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.handlers = append(l.handlers, handlers...)
}

func (l *logger) Flush() {
	for _, handler := range l.handlers {
		handler.Flush()
	}
}
func (l *logger) Close() {
	for _, handler := range l.handlers {
		handler.Close()
	}
}

func (l *logger) Trace(format string, v ...any) { l.output(TraceLevel, nil, format, v...) }
func (l *logger) Debug(format string, v ...any) { l.output(DebugLevel, nil, format, v...) }
func (l *logger) Info(format string, v ...any)  { l.output(InfoLevel, nil, format, v...) }
func (l *logger) Warn(format string, v ...any)  { l.output(WarnLevel, nil, format, v...) }
func (l *logger) Error(format string, v ...any) { l.output(ErrorLevel, nil, format, v...) }
func (l *logger) Fatal(format string, v ...any) { l.output(FatalLevel, nil, format, v...) }
func (l *logger) Panic(format string, v ...any) { l.output(PanicLevel, nil, format, v...) }

func (l *logger) CtxTrace(ctx context.Context, format string, v ...any) {
	l.output(TraceLevel, ctx, format, v...)
}
func (l *logger) CtxDebug(ctx context.Context, format string, v ...any) {
	l.output(DebugLevel, ctx, format, v...)
}
func (l *logger) CtxInfo(ctx context.Context, format string, v ...any) {
	l.output(InfoLevel, ctx, format, v...)
}
func (l *logger) CtxWarn(ctx context.Context, format string, v ...any) {
	l.output(WarnLevel, ctx, format, v...)
}
func (l *logger) CtxError(ctx context.Context, format string, v ...any) {
	l.output(ErrorLevel, ctx, format, v...)
}
func (l *logger) CtxFatal(ctx context.Context, format string, v ...any) {
	l.output(FatalLevel, ctx, format, v...)
}
func (l *logger) CtxPanic(ctx context.Context, format string, v ...any) {
	l.output(PanicLevel, ctx, format, v...)
}

func (l *logger) output(level Level, ctx context.Context, format string, v ...any) {
	for _, handler := range l.handlers {
		handler.Output(level, ctx, format, v...)
	}
}

// func init() { go defaultLogger.(*logger).serve() }
// func (l *logger) serve() {
// 	for msg := range l.ch {
// 		_, _ = l.out.Write([]byte(msg))
// 	}
// }
