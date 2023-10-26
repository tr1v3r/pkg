package log

import (
	"context"
	"io"
	"sync"
)

var (
	defaultHandler Handler = NewStreamHandler(InfoLevel)
	defaultLogger  Logger  = NewLogger(defaultHandler)
)

// NewLogger new logger
func NewLogger(handlers ...Handler) Logger { return &logger{handlers: handlers} }

// Logger logger interface
type Logger interface {
	RegisterHandler(...Handler)
	ClearHandler()

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
	io.Writer

	SetLevel(Level)
	Output(level Level, ctx context.Context, format string, v ...any)

	Flush()
	Close()

	RegisterOutput(io.Writer)
	SetOutput(io.Writer)
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

func (l *logger) ClearHandler() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.handlers = make([]Handler, 0, 4)
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

func (l *logger) Trace(format string, v ...any) { l.CtxTrace(nil, format, v...) } // nolint
func (l *logger) Debug(format string, v ...any) { l.CtxDebug(nil, format, v...) } // nolint
func (l *logger) Info(format string, v ...any)  { l.CtxInfo(nil, format, v...) }  // nolint
func (l *logger) Warn(format string, v ...any)  { l.CtxWarn(nil, format, v...) }  // nolint
func (l *logger) Error(format string, v ...any) { l.CtxError(nil, format, v...) } // nolint
func (l *logger) Fatal(format string, v ...any) { l.CtxFatal(nil, format, v...) } // nolint
func (l *logger) Panic(format string, v ...any) { l.CtxPanic(nil, format, v...) } // nolint

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
