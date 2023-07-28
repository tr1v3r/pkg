package log

import (
	"context"
	"fmt"
	"io"
	"os"
)

var defaultLogger = NewLogger()

// NewLogger new logger
func NewLogger() Logger {
	return &logger{
		Formatter: NewFormatter(true),

		level: InfoLevel,
		out:   os.Stdout,
	}
}

// Logger logger interface
type Logger interface {
	SetLevel(level Level)
	SetOutput(out io.Writer)

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
	out   io.Writer
}

// SetLevel set log level
func (l *logger) SetLevel(level Level)        { l.level = level }
func (l *logger) allowLevel(level Level) bool { return level <= l.level }

// SetOutput set output writer
func (l *logger) SetOutput(out io.Writer) { l.out = out }

func (l *logger) Trace(format string, v ...any) { _, _ = l.output(TraceLevel, nil, format, v...) }
func (l *logger) Debug(format string, v ...any) { _, _ = l.output(DebugLevel, nil, format, v...) }
func (l *logger) Info(format string, v ...any)  { _, _ = l.output(InfoLevel, nil, format, v...) }
func (l *logger) Warn(format string, v ...any)  { _, _ = l.output(WarnLevel, nil, format, v...) }
func (l *logger) Error(format string, v ...any) { _, _ = l.output(ErrorLevel, nil, format, v...) }
func (l *logger) Fatal(format string, v ...any) { _, _ = l.output(FatalLevel, nil, format, v...) }
func (l *logger) Panic(format string, v ...any) { _, _ = l.output(PanicLevel, nil, format, v...) }

func (l *logger) CtxTrace(ctx context.Context, format string, v ...any) {
	_, _ = l.output(TraceLevel, nil, format, v...)
}
func (l *logger) CtxDebug(ctx context.Context, format string, v ...any) {
	_, _ = l.output(DebugLevel, nil, format, v...)
}
func (l *logger) CtxInfo(ctx context.Context, format string, v ...any) {
	_, _ = l.output(InfoLevel, nil, format, v...)
}
func (l *logger) CtxWarn(ctx context.Context, format string, v ...any) {
	_, _ = l.output(WarnLevel, nil, format, v...)
}
func (l *logger) CtxError(ctx context.Context, format string, v ...any) {
	_, _ = l.output(ErrorLevel, nil, format, v...)
}
func (l *logger) CtxFatal(ctx context.Context, format string, v ...any) {
	_, _ = l.output(FatalLevel, nil, format, v...)
}
func (l *logger) CtxPanic(ctx context.Context, format string, v ...any) {
	_, _ = l.output(PanicLevel, nil, format, v...)
}

func (l *logger) getLogID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	return ctx.Value("log_id").(string)
}

func (l *logger) output(level Level, ctx context.Context, format string, v ...any) (int, error) {
	if !l.allowLevel(level) {
		return 0, nil
	}
	return l.out.Write([]byte(fmt.Sprintf(l.Format(level, l.getLogID(ctx), format), v...)))
}
