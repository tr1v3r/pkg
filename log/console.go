package log

import (
	"io"
	"os"
)

// Console returns a Sink that writes colored text to stdout.
func Console(opts ...SinkOption) *Sink {
	return newSink(
		NewTextEncoder(true),
		os.Stdout,
		opts...,
	)
}

// ConsoleTo returns a Sink that writes colored text to the given writer.
func ConsoleTo(w io.Writer, opts ...SinkOption) *Sink {
	return newSink(
		NewTextEncoder(true),
		w,
		opts...,
	)
}
