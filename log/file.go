package log

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// NewFileLogger 滚动文件日志
func NewFileLogger(dir string, opts ...FileLoggerOption) (*FileLogger, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("find file abs path fail: %w", err)
	}
	if f, err := os.Stat(dir); err != nil || !f.IsDir() {
		return nil, fmt.Errorf("set log dir fail: %w", err)
	}

	logger := &FileLogger{
		Dir:           dir,
		Formatter:     func(log []byte) []byte { return log },
		intervalLevel: IntervalHour,
	}
	for _, opt := range opts {
		logger = opt(logger)
	}
	return logger, nil
}

// FileLoggerOption ...
type FileLoggerOption func(*FileLogger) *FileLogger

var (
	// // FileLoggerLevel set log info level
	// FileLoggerLevel = func(level Level) FileLoggerOption {
	// 	return func(logger *FileLogger) *FileLogger {
	// 		logger.Level = level
	// 		return logger
	// 	}
	// }

	// FileLoggerInterval set file interval: minute hour day
	FileLoggerInterval = func(interval time.Duration) FileLoggerOption {
		return func(logger *FileLogger) *FileLogger {
			switch {
			case interval <= time.Minute:
				logger.intervalLevel = IntervalMinute
			case interval <= time.Hour:
				logger.intervalLevel = IntervalHour
			default:
				logger.intervalLevel = IntervalDay
			}
			return logger
		}
	}

	// FileLoggerFormatter set log formatter
	FileLoggerFormatter = func(formatter func(log []byte) (formattedLog []byte)) FileLoggerOption {
		return func(logger *FileLogger) *FileLogger {
			logger.Formatter = formatter
			return logger
		}
	}
)

// IntervalLevel ...
type IntervalLevel string

const (
	// IntervalMinute ...
	IntervalMinute IntervalLevel = "minute"
	// IntervalHour ...
	IntervalHour IntervalLevel = "hour"
	// IntervalDay ...
	IntervalDay IntervalLevel = "day"
)

// FileLogger file logger
type FileLogger struct {
	Dir       string
	Formatter func(log []byte) []byte

	// Level Level

	intervalLevel IntervalLevel

	mu     sync.RWMutex
	output *os.File
}

func (f *FileLogger) Write(p []byte) (int, error) {
	output, err := f.getOutput()
	if err != nil {
		return 0, fmt.Errorf("get output log file handler fail: %w", err)
	}
	return output.Write(f.Formatter(p))
}

func (f *FileLogger) FileName() string {
	fileName := bytes.NewBuffer([]byte(f.Dir))
	fileName.WriteByte('/')

	now := time.Now()
	fileName.WriteString(now.Format("2006-01-02T"))

	switch f.intervalLevel {
	case IntervalMinute:
		fileName.WriteByte('_')
		fileName.WriteString(fmt.Sprintf("%02d", now.Hour()))
		fileName.WriteByte('_')
		fileName.WriteString(fmt.Sprintf("%02d", now.Minute()))
	case IntervalHour:
		fileName.WriteByte('_')
		fileName.WriteString(fmt.Sprintf("%02d", now.Hour()))
	case IntervalDay:
	}
	fileName.WriteString(".log")
	return fileName.String()
}

func (f *FileLogger) file() (*os.File, error) {
	logFileName := f.FileName()
	_f, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file %s fail: %w", logFileName, err)
	}
	return _f, nil
}

func (f *FileLogger) getOutput() (*os.File, error) {
	f.mu.RLock()
	if f.output != nil && f.output.Name() == f.FileName() {
		defer f.mu.RUnlock()
		return f.output, nil
	}
	if f.output != nil {
		f.output.Close()
	}
	f.mu.RUnlock()

	output, err := f.file()
	if err != nil {
		return nil, err
	}

	f.mu.Lock()
	f.output = output
	f.mu.Unlock()

	return output, nil
}
