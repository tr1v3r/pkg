package log

import (
	"context"
	"io"

	"github.com/sirupsen/logrus"
)

type Level = logrus.Level
type Formatter = logrus.Formatter

const (
	PanicLevel Level = logrus.PanicLevel
	FatalLevel Level = logrus.FatalLevel
	ErrorLevel Level = logrus.ErrorLevel
	WarnLevel  Level = logrus.WarnLevel
	InfoLevel  Level = logrus.InfoLevel
	DebugLevel Level = logrus.DebugLevel
	TraceLevel Level = logrus.TraceLevel
)

var defaultLogger = logrus.New()

func init() {
	defaultLogger.SetFormatter(&logrus.TextFormatter{
		ForceQuote:      true,                  //键值对加引号
		TimestampFormat: "2006-01-02 15:04:05", //时间格式
		FullTimestamp:   true,
	})
}

// SetFormatter set formatter
func SetFormatter(formatter Formatter) {
	defaultLogger.SetFormatter(formatter)
}

// SetOutput set output
func SetOutput(out io.Writer) {
	defaultLogger.SetOutput(out)
}

// SetLevel set output log level
func SetLevel(l Level) {
	defaultLogger.SetLevel(l)
}

// Trace ...
func Trace(format string, args ...interface{}) {
	defaultLogger.Tracef(format, args...)
}

// CtxTrace ...
func CtxTrace(_ context.Context, format string, args ...interface{}) {
	defaultLogger.Tracef(format, args...)
}

// Debug ...
func Debug(format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}

// CtxDebug ...
func CtxDebug(_ context.Context, format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}

// Info ...
func Info(format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}

// CtxInfo ...
func CtxInfo(_ context.Context, format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}

// Warn ...
func Warn(format string, args ...interface{}) {
	defaultLogger.Warnf(format, args...)
}

// CtxWarn ...
func CtxWarn(_ context.Context, format string, args ...interface{}) {
	defaultLogger.Warnf(format, args...)
}

// Error ...
func Error(format string, args ...interface{}) {
	defaultLogger.Errorf(format, args...)
}

// CtxError ...
func CtxError(_ context.Context, format string, args ...interface{}) {
	defaultLogger.Errorf(format, args...)
}

// Fatal ...
func Fatal(format string, args ...interface{}) {
	defaultLogger.Fatalf(format, args...)
}

// CtxFatal ...
func CtxFatal(_ context.Context, format string, args ...interface{}) {
	defaultLogger.Fatalf(format, args...)
}

// Panic ...
func Panic(format string, args ...interface{}) {
	defaultLogger.Panicf(format, args...)
}

// CtxPanic ...
func CtxPanic(_ context.Context, format string, args ...interface{}) {
	defaultLogger.Panicf(format, args...)
}
