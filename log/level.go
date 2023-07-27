package log

// Level log level
type Level uint32

const (
	// PanicLevel panic level
	PanicLevel Level = iota
	// FatalLevel fatal level
	FatalLevel
	// ErrorLevel error level
	ErrorLevel
	// WarnLevel warn level
	WarnLevel
	// InfoLevel info level
	InfoLevel
	// DebugLevel debug level
	DebugLevel
	// TraceLevel trace level
	TraceLevel
)

func (l Level) String() string {
	switch l {
	case TraceLevel:
		return "Trace"
	case DebugLevel:
		return "Debug"
	case InfoLevel:
		return "Info"
	case WarnLevel:
		return "Warn"
	case ErrorLevel:
		return "Error"
	case FatalLevel:
		return "Fatal"
	case PanicLevel:
		return "Panic"
	default:
		return ""
	}
}

func (l Level) Color() int {
	switch l {
	case TraceLevel:
		return 45
	case DebugLevel:
		return 39
	case InfoLevel:
		return 33
	case WarnLevel:
		return 148
	case ErrorLevel:
		return 161
	case FatalLevel:
		return 160
	case PanicLevel:
		return 196
	default:
		return 0
	}
}
