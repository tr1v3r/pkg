package log

import "io"

// Encoder converts a Record to bytes.
type Encoder interface {
	Encode(record Record) []byte
}

// SinkOption configures a Sink.
type SinkOption func(*Sink)

// WithLevel sets the minimum log level for this sink.
func WithLevel(level Level) SinkOption {
	return func(s *Sink) { s.level = level }
}

// WithAsync enables async (buffered) writing with the given channel size.
func WithAsync(bufSize int) SinkOption {
	return func(s *Sink) { s.ch = make(chan []byte, bufSize) }
}

// Sink is a self-contained output unit: level filter + encoder + writer.
type Sink struct {
	level  Level
	enc    Encoder
	writer io.Writer
	closer io.Closer
	ch     chan []byte // nil = sync mode
	quit   chan struct{}
}

// newSink creates a Sink with the given encoder, writer, and options.
func newSink(enc Encoder, w io.Writer, opts ...SinkOption) *Sink {
	s := &Sink{
		level:  InfoLevel,
		enc:    enc,
		writer: w,
		quit:   make(chan struct{}),
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.ch != nil {
		go s.drain()
	}
	return s
}

// Write encodes the record and writes it.
// Records below the sink's level are dropped.
func (s *Sink) Write(record Record) {
	if record.Level < s.level {
		return
	}
	data := s.enc.Encode(record)
	if s.ch != nil {
		select {
		case s.ch <- data:
		case <-s.quit:
		}
	} else {
		s.writer.Write(data)
	}
}

// SetLevel changes the minimum level at runtime.
func (s *Sink) SetLevel(level Level) { s.level = level }

// Sync flushes buffered data.
func (s *Sink) Sync() {
	if s.ch == nil {
		if f, ok := s.writer.(interface{ Sync() error }); ok {
			f.Sync()
		}
		return
	}
	for i := 0; i < cap(s.ch); i++ {
		select {
		case data := <-s.ch:
			s.writer.Write(data)
		default:
			return
		}
	}
}

// Close stops the sink and releases resources.
func (s *Sink) Close() error {
	close(s.quit)
	s.Sync()
	if s.closer != nil {
		return s.closer.Close()
	}
	return nil
}

func (s *Sink) drain() {
	for data := range s.ch {
		s.writer.Write(data)
	}
}
