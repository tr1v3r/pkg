package log

// Level represents the severity level of a log message
// Levels are ordered from most verbose to most severe
//
// Example usage:
//   log.SetLevel(log.InfoLevel)
//   log.Debug("This won't be logged") // filtered out
//   log.Info("This will be logged")   // logged

type Level uint32

const (
	// TraceLevel is the most verbose level, typically used for detailed debugging
	TraceLevel Level = iota

	// DebugLevel is used for debugging information that's useful during development
	DebugLevel

	// InfoLevel is used for general operational information
	InfoLevel

	// WarnLevel is used for warning conditions that don't require immediate action
	WarnLevel

	// ErrorLevel is used for error conditions that require attention
	ErrorLevel

	// FatalLevel is used for fatal errors that cause the program to exit
	FatalLevel

	// PanicLevel is the most severe level, causing the program to panic
	PanicLevel
)

// String returns the string representation of the log level
func (l Level) String() string {
	switch l {
	case TraceLevel:
		return "TRACE"
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	case PanicLevel:
		return "PANIC"
	default:
		return "UNKNOWN"
	}
}

// Color returns the ANSI color code for the log level
// Used for colored console output
func (l Level) Color() int {
	switch l {
	case TraceLevel:
		return 45 // Magenta
	case DebugLevel:
		return 39 // Blue
	case InfoLevel:
		return 33 // Yellow
	case WarnLevel:
		return 148 // Orange
	case ErrorLevel:
		return 161 // Red
	case FatalLevel:
		return 160 // Bright Red
	case PanicLevel:
		return 196 // Brightest Red
	default:
		return 0 // Default
	}
}

// ParseLevel parses a string into a Level
// Returns InfoLevel for unknown strings
func ParseLevel(s string) Level {
	switch s {
	case "trace", "TRACE":
		return TraceLevel
	case "debug", "DEBUG":
		return DebugLevel
	case "info", "INFO":
		return InfoLevel
	case "warn", "WARN", "warning", "WARNING":
		return WarnLevel
	case "error", "ERROR":
		return ErrorLevel
	case "fatal", "FATAL":
		return FatalLevel
	case "panic", "PANIC":
		return PanicLevel
	default:
		return InfoLevel
	}
}