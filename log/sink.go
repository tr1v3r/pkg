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

// WithSync disables async writing, overriding a default async sink.
func WithSync() SinkOption {
	return func(s *Sink) { s.ch = nil }
}

// Sink is a self-contained output unit: level filter + encoder + writer.
type Sink struct {
	level  Level
	enc    Encoder
	writer io.Writer
	closer io.Closer
	ch     chan []byte // nil = sync mode
	done   chan struct{}
}

// newSink creates a Sink with the given encoder, writer, and options.
func newSink(enc Encoder, w io.Writer, opts ...SinkOption) *Sink {
	s := &Sink{
		level:  InfoLevel,
		enc:    enc,
		writer: w,
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.ch != nil {
		s.done = make(chan struct{})
		go s.drain()
	}
	return s
}

// Log encodes the record and writes it.
// Records below the sink's level are dropped.
func (s *Sink) Log(record Record) {
	if record.Level < s.level {
		return
	}
	data := s.enc.Encode(record)
	if s.ch != nil {
		s.ch <- data
	} else {
		s.writer.Write(data)
	}
}

// Write implements io.Writer, writing raw bytes directly to the underlying writer.
// This allows a Sink to replace gin.DefaultWriter or any io.Writer destination.
func (s *Sink) Write(p []byte) (int, error) {
	if s.ch != nil {
		cp := make([]byte, len(p))
		copy(cp, p)
		s.ch <- cp
		return len(p), nil
	}
	return s.writer.Write(p)
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
	// Block until drain goroutine has consumed everything in the channel.
	// Send a sentinel nil to act as a barrier: drain will process all
	// pending data before reaching the nil, then we know we're flushed.
	s.ch <- nil
	<-s.done // wait for drain to signal barrier passed
	// Restart drain for future writes.
	s.done = make(chan struct{})
	go s.drain()
}

// Close stops the sink and releases resources.
func (s *Sink) Close() error {
	if s.ch != nil {
		close(s.ch)
		<-s.done
	}
	if s.closer != nil {
		return s.closer.Close()
	}
	return nil
}

func (s *Sink) drain() {
	for data := range s.ch {
		if data == nil {
			s.done <- struct{}{}
			return
		}
		s.writer.Write(data)
	}
	s.done <- struct{}{}
}
