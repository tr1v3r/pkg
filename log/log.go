package log

import "github.com/sirupsen/logrus"

var logger = logrus.New()

func init() {
	logger.SetFormatter(&logrus.TextFormatter{
		ForceQuote:      true,                  //键值对加引号
		TimestampFormat: "2006-01-02 15:04:05", //时间格式
		FullTimestamp:   true,
	})
}

// Trace ...
func Trace(format string, args ...interface{}) {
	logger.Tracef(format, args...)
}

// Debug ...
func Debug(format string, args ...interface{}) {
	logger.Debugf(format, args...)
}

// Info ...
func Info(format string, args ...interface{}) {
	logger.Infof(format, args...)
}

// Warn ...
func Warn(format string, args ...interface{}) {
	logger.Warnf(format, args...)
}

// Error ...
func Error(format string, args ...interface{}) {
	logger.Errorf(format, args...)
}

// Fatal ...
func Fatal(format string, args ...interface{}) {
	logger.Fatalf(format, args...)
}

// Panic ...
func Panic(format string, args ...interface{}) {
	logger.Panicf(format, args...)
}
