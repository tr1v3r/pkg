package log

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
)

// NewStreamHandler create new stream handler
func NewStreamHandler(level Level) Handler {
	handler := &StreamHandler{
		Formatter: NewStreamFormatter(true),

		level: level,
		ch:    make(chan []byte, 8*1024),
		out:   os.Stdout,
	}
	go handler.serve()
	return handler
}

// StreamHandler stream log handler
type StreamHandler struct {
	Formatter

	level Level
	ch    chan []byte
	out   io.Writer
}

func (s *StreamHandler) SetLevel(level Level)        { s.level = level }
func (s *StreamHandler) allowLevel(level Level) bool { return level >= s.level }

func (s *StreamHandler) SetOutput(out io.Writer)      { s.out = out }
func (s *StreamHandler) RegisterOutput(out io.Writer) { s.out = io.MultiWriter(s.out, out) }

func (s *StreamHandler) Output(level Level, ctx context.Context, format string, v ...any) {
	if s.allowLevel(level) {
		s.ch <- []byte(fmt.Sprintf(s.Format(level, ctx, format), v...))
	}
}

func (s *StreamHandler) Write(p []byte) (int, error) { return s.out.Write(p) }

func (s *StreamHandler) Flush() {
	runtime.Gosched()
	for {
		select {
		case msg := <-s.ch:
			_, _ = s.out.Write(msg)
		default:
			return
		}
	}
}
func (s *StreamHandler) Close() {
	close(s.ch)
	s.Flush()
}

// func init() { go defaultLogger.(*logger).serve() }
func (s *StreamHandler) serve() {
	for msg := range s.ch {
		_, _ = s.out.Write(msg)
	}
}
