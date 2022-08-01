package log

import (
	"io"

	"github.com/sirupsen/logrus"
)

type Level = logrus.Level
type Formatter = logrus.Formatter

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

// Trace ...
func Trace(format string, args ...interface{}) {
	defaultLogger.Tracef(format, args...)
}

// Debug ...
func Debug(format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}

// Info ...
func Info(format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}

// Warn ...
func Warn(format string, args ...interface{}) {
	defaultLogger.Warnf(format, args...)
}

// Error ...
func Error(format string, args ...interface{}) {
	defaultLogger.Errorf(format, args...)
}

// Fatal ...
func Fatal(format string, args ...interface{}) {
	defaultLogger.Fatalf(format, args...)
}

// Panic ...
func Panic(format string, args ...interface{}) {
	defaultLogger.Panicf(format, args...)
}
