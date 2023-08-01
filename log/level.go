package log

// Level log level
type Level uint32

const (
	// TraceLevel trace level
	TraceLevel Level = iota
	// DebugLevel debug level
	DebugLevel
	// InfoLevel info level
	InfoLevel
	// WarnLevel warn level
	WarnLevel
	// ErrorLevel error level
	ErrorLevel
	// FatalLevel fatal level
	FatalLevel
	// PanicLevel panic level
	PanicLevel
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
