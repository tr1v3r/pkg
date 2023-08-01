package log

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
)

var defaultLogger = NewLogger()

func init() { go defaultLogger.(*logger).serve() }

// NewLogger new logger
func NewLogger() Logger {
	return &logger{
		Formatter: NewFormatter(true),

		level: InfoLevel,
		ch:    make(chan string, 8*1024),
		out:   os.Stdout,
	}
}

// Logger logger interface
type Logger interface {
	SetLevel(level Level)
	SetOutput(out io.Writer)

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

type logger struct {
	*Formatter

	level Level
	ch    chan string
	out   io.Writer
}

func (l *logger) Flush() {
	runtime.Gosched()
	for {
		select {
		case msg := <-l.ch:
			_, _ = l.out.Write([]byte(msg))
		default:
			return
		}
	}
}

func (l *logger) Close() {
	close(l.ch)
	l.Flush()
}

// SetLevel set log level
func (l *logger) SetLevel(level Level)        { l.level = level }
func (l *logger) allowLevel(level Level) bool { return level >= l.level }

// SetOutput set output writer
func (l *logger) SetOutput(out io.Writer) { l.out = out }

func (l *logger) Trace(format string, v ...any) { l.output(TraceLevel, nil, format, v...) }
func (l *logger) Debug(format string, v ...any) { l.output(DebugLevel, nil, format, v...) }
func (l *logger) Info(format string, v ...any)  { l.output(InfoLevel, nil, format, v...) }
func (l *logger) Warn(format string, v ...any)  { l.output(WarnLevel, nil, format, v...) }
func (l *logger) Error(format string, v ...any) { l.output(ErrorLevel, nil, format, v...) }
func (l *logger) Fatal(format string, v ...any) { l.output(FatalLevel, nil, format, v...) }
func (l *logger) Panic(format string, v ...any) { l.output(PanicLevel, nil, format, v...) }

func (l *logger) CtxTrace(ctx context.Context, format string, v ...any) {
	l.output(TraceLevel, nil, format, v...)
}
func (l *logger) CtxDebug(ctx context.Context, format string, v ...any) {
	l.output(DebugLevel, nil, format, v...)
}
func (l *logger) CtxInfo(ctx context.Context, format string, v ...any) {
	l.output(InfoLevel, nil, format, v...)
}
func (l *logger) CtxWarn(ctx context.Context, format string, v ...any) {
	l.output(WarnLevel, nil, format, v...)
}
func (l *logger) CtxError(ctx context.Context, format string, v ...any) {
	l.output(ErrorLevel, nil, format, v...)
}
func (l *logger) CtxFatal(ctx context.Context, format string, v ...any) {
	l.output(FatalLevel, nil, format, v...)
}
func (l *logger) CtxPanic(ctx context.Context, format string, v ...any) {
	l.output(PanicLevel, nil, format, v...)
}

func (l *logger) getLogID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	return ctx.Value("log_id").(string)
}

func (l *logger) output(level Level, ctx context.Context, format string, v ...any) {
	if !l.allowLevel(level) {
		return
	}
	l.ch <- fmt.Sprintf(l.Format(level, l.getLogID(ctx), format), v...)
}

func (l *logger) serve() {
	for msg := range l.ch {
		_, _ = l.out.Write([]byte(msg))
	}
}
