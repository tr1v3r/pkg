package log

import (
	"fmt"
	"os"
)

// File returns a Sink that writes plain text to a single file.
// The file is created if it doesn't exist, appended to if it does.
func File(path string, opts ...SinkOption) (*Sink, error) {
	//nolint:gosec // G703: path is caller-supplied by design, same as os.OpenFile
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("log: open file %s: %w", path, err)
	}
	opts = append([]SinkOption{WithAsync(1024)}, opts...)
	s := newSink(NewTextEncoder(false), f, opts...)
	s.closer = f
	return s, nil
}
