package log

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// NewFileHandler 滚动文件日志
func NewFileHandler(level Level, dir string, opts ...FileHandlerOption) (Handler, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("find file abs path fail: %w", err)
	}
	if f, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			if os.MkdirAll(dir, 0755) != nil {
				return nil, fmt.Errorf("create dir fail: %w", err)
			}
		} else if !f.IsDir() {
			return nil, fmt.Errorf("set log dir fail: %w", err)
		}
	}

	handler := &FileHandler{
		Formatter: NewStreamFormatter(true),

		Dir: dir,

		level:         level,
		ch:            make(chan []byte, 8*1024),
		intervalLevel: IntervalHour,
	}
	for _, opt := range opts {
		handler = opt(handler)
	}
	return handler, nil
}

// FileHandlerOption ...
type FileHandlerOption func(*FileHandler) *FileHandler

var (
	// FileHandlerInterval set file interval: minute hour day
	FileHandlerInterval = func(interval time.Duration) FileHandlerOption {
		return func(handler *FileHandler) *FileHandler {
			switch {
			case interval <= time.Minute:
				handler.intervalLevel = IntervalMinute
			case interval <= time.Hour:
				handler.intervalLevel = IntervalHour
			default:
				handler.intervalLevel = IntervalDay
			}
			return handler
		}
	}

	// FileHandlerFormatter set file formatter
	FileHandlerFormatter = func(f Formatter) FileHandlerOption {
		return func(handler *FileHandler) *FileHandler {
			handler.Formatter = f
			return handler
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

// FileHandler file handler
type FileHandler struct {
	Formatter

	Dir string

	intervalLevel IntervalLevel

	level Level
	ch    chan []byte

	mu  sync.RWMutex
	out *os.File

	once sync.Once
}

func (f *FileHandler) SetLevel(level Level)        { f.level = level }
func (f *FileHandler) allowLevel(level Level) bool { return level >= f.level }

func (f *FileHandler) SetOutput(out io.Writer)      { /* do nothing */ }
func (f *FileHandler) RegisterOutput(out io.Writer) { /* do nothing */ }

func (f *FileHandler) Output(level Level, ctx context.Context, format string, v ...any) {
	f.once.Do(func() { go f.serve() })
	if f.allowLevel(level) {
		f.ch <- []byte(fmt.Sprintf(f.Format(level, ctx, format), v...))
	}
}

func (f *FileHandler) Write(p []byte) (int, error) {
	output, err := f.getOutput()
	if err != nil {
		return 0, fmt.Errorf("get file output writer fail: %w", err)
	}
	return output.Write(p)
}

func (f *FileHandler) FileName() string {
	fileName := bytes.NewBuffer([]byte(f.Dir))
	fileName.WriteByte('/')

	switch f.intervalLevel {
	case IntervalMinute:
		fileName.WriteString(time.Now().Format("2006-01-02T_15_04"))
	case IntervalHour:
		fileName.WriteString(time.Now().Format("2006-01-02T_15"))
	case IntervalDay:
		fileName.WriteString(time.Now().Format("2006-01-02"))
	}
	fileName.WriteString(".log")

	return fileName.String()
}

func (f *FileHandler) file() (*os.File, error) {
	logFileName := f.FileName()
	curFile, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file %s fail: %w", logFileName, err)
	}
	return curFile, nil
}

func (f *FileHandler) getOutput() (io.Writer, error) {
	if out := f.checkOutput(); out != nil {
		return out, nil
	}

	// create new file output writer
	output, err := f.file()
	if err != nil {
		return nil, err
	}

	f.mu.Lock()
	f.out = output
	f.mu.Unlock()

	return output, nil
}

// checkOutput return f.out if valid, else return nil
func (f *FileHandler) checkOutput() *os.File {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// close file handler when filename not matched
	if f.out != nil && f.out.Name() != f.FileName() {
		f.out.Close()
		return nil
	}
	return f.out // f.out is nil or f.out match file name
}

func (f *FileHandler) Flush() {
	runtime.Gosched()
	for {
		select {
		case msg := <-f.ch:
			_, _ = f.Write(msg)
		default:
			return
		}
	}
}
func (f *FileHandler) Close() {
	close(f.ch)
	f.Flush()
}

func (f *FileHandler) serve() {
	for msg := range f.ch {
		_, _ = f.Write(msg)
	}
}
