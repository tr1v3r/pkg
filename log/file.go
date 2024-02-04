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

	"golang.org/x/time/rate"
)

var _ Handler = (*FileHandler)(nil)

// NewFileHandler 滚动文件日志
func NewFileHandler(level Level, dir string, opts ...FileHandlerOption) (*FileHandler, error) {
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

		closed: make(chan struct{}),

		limiter: rate.NewLimiter(100, 1000),
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
			case interval <= 0:
				handler.intervalLevel = IntervalNone
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
	// FileHandlerLogFilePrefix set log file prefix
	FileHandlerLogFilePrefix = func(prefix string) FileHandlerOption {
		return func(handler *FileHandler) *FileHandler {
			handler.filePrefix = prefix + "."
			return handler
		}
	}
)

// IntervalLevel set interval level
type IntervalLevel string

const (
	// IntervalNone creat log file without interval
	IntervalNone IntervalLevel = "none"
	// IntervalMinute create log file every minute
	IntervalMinute IntervalLevel = "minute"
	// IntervalHour create log file every hour
	IntervalHour IntervalLevel = "hour"
	// IntervalDay create log file every day
	IntervalDay IntervalLevel = "day"
)

// FileHandler file handler
type FileHandler struct {
	Formatter

	Dir string

	intervalLevel IntervalLevel

	filePrefix string

	level Level
	ch    chan []byte

	mu  sync.RWMutex
	out *os.File

	closeOnce sync.Once
	closed    chan struct{}

	once    sync.Once
	limiter *rate.Limiter
}

func (f *FileHandler) SetLevel(level Level)        { f.level = level }
func (f *FileHandler) allowLevel(level Level) bool { return level >= f.level }

func (f *FileHandler) SetOutput(out io.Writer)      { /* do nothing */ }
func (f *FileHandler) RegisterOutput(out io.Writer) { /* do nothing */ }

func (f *FileHandler) Output(level Level, ctx context.Context, format string, v ...any) {
	f.once.Do(func() {
		_ = f.refreshWriter()
		go f.serve()
	})
	if f.allowLevel(level) {
		f.ch <- []byte(fmt.Sprintf(f.Format(level, ctx, format), v...))
	}
}

func (f *FileHandler) Write(p []byte) (int, error) {
	if err := f.refreshWriter(); err != nil {
		return 0, fmt.Errorf("refresh writer fail: %w", err)
	}

	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.out.Write(p)
}

func (f *FileHandler) FileName() string {
	fileName := bytes.NewBuffer([]byte(f.Dir))
	fileName.WriteByte('/')

	fileName.WriteString(f.filePrefix)

	now := time.Now()
	switch f.intervalLevel {
	case IntervalMinute:
		fileName.WriteString(now.Format("2006-01-02T_15_04."))
	case IntervalHour:
		fileName.WriteString(now.Format("2006-01-02T_15."))
	case IntervalDay:
		fileName.WriteString(now.Format("2006-01-02."))
	}
	fileName.WriteString("log")

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

// checkOutput return f.out if valid, else return nil
func (f *FileHandler) needRefreshWriter() bool {
	if !f.limiter.Allow() {
		return false
	}

	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.out == nil || f.out.Name() != f.FileName()
}

func (f *FileHandler) refreshWriter() error {
	if !f.needRefreshWriter() {
		return nil
	}

	// create new file output writer
	output, err := f.file()
	if err != nil {
		return err
	}

	f.mu.Lock()
	if o := f.out; o != nil {
		go o.Close()
	}
	f.out = output
	f.mu.Unlock()

	return nil
}

func (f *FileHandler) Flush() {
	runtime.Gosched()
	for i := cap(f.ch); i > 0; i-- { // max try times less than capacity of ch
		select {
		case msg, ok := <-f.ch:
			if !ok {
				f.close() // in case serve goroutine not running
				return
			}
			_, _ = f.Write(msg)
		default:
			f.flush()
			return
		}
	}
}

func (f *FileHandler) flush() {
	f.mu.RLock()
	defer f.mu.RUnlock()
	_ = f.out.Sync()
}

func (f *FileHandler) Close() {
	f.close()
	f.Flush()
	<-f.closed
}

func (f *FileHandler) close() {
	f.closeOnce.Do(func() { close(f.closed) })
}

func (f *FileHandler) serve() {
	for msg := range f.ch {
		_, _ = f.Write(msg)
	}
	f.close()
}
