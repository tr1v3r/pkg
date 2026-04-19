package log

import (
	"fmt"
	"os"
)

// File returns a Sink that writes plain text to a single file.
// The file is created if it doesn't exist, appended to if it does.
func File(path string, opts ...SinkOption) (*Sink, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("log: open file %s: %w", path, err)
	}
	s := newSink(NewTextEncoder(false), f, opts...)
	s.closer = f
	return s, nil
}
