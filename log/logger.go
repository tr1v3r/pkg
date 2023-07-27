package log

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

var defaultLogger = NewLogger()

// NewLogger new logger
func NewLogger() Logger {
	return &logger{
		level: InfoLevel,
		out:   os.Stdout,
		color: true,
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
	level Level
	out   io.Writer
	color bool
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

func (l *logger) format(level Level, logID string, format string) string {
	if l.color {
		return fmt.Sprintf("\033[38;5;%dm%s [%s]%s\033[0m\n", level.Color(), time.Now().Format(time.RFC3339), strings.ToUpper(level.String()), format)
	}
	return fmt.Sprintf("%s [%s]%s\n", time.Now().Format(time.RFC3339), strings.ToUpper(level.String()), format)
}

func (l *logger) output(level Level, ctx context.Context, format string, v ...any) (int, error) {
	if !l.allowLevel(level) {
		return 0, nil
	}
	return l.out.Write([]byte(fmt.Sprintf(l.format(level, l.getLogID(ctx), format), v...)))
}
