package log

import (
	"testing"
)

func TestOutput(t *testing.T) {
	SetLevel(TraceLevel)

	Trace("trace message")
	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")
	Fatal("fatal message")
	Panic("panic message")

	Flush()
}
