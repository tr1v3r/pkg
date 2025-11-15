package slog

import (
	"github.com/tr1v3r/pkg/log"
	stdslog "log/slog"
)

// ToSlogLevel converts log package Level to standard slog.Level
func ToSlogLevel(level log.Level) stdslog.Level {
	switch level {
	case log.TraceLevel:
		return stdslog.LevelDebug - 4 // Custom trace level below debug
	case log.DebugLevel:
		return stdslog.LevelDebug
	case log.InfoLevel:
		return stdslog.LevelInfo
	case log.WarnLevel:
		return stdslog.LevelWarn
	case log.ErrorLevel:
		return stdslog.LevelError
	case log.FatalLevel:
		return stdslog.LevelError + 4 // Custom fatal level above error
	case log.PanicLevel:
		return stdslog.LevelError + 8 // Custom panic level above error
	default:
		return stdslog.LevelInfo
	}
}

// FromSlogLevel converts standard slog.Level to log package Level
func FromSlogLevel(level stdslog.Level) log.Level {
	// Custom trace levels (below debug)
	if level <= stdslog.LevelDebug-4 {
		return log.TraceLevel
	}

	// Debug levels (including custom between trace and debug)
	if level <= stdslog.LevelDebug-1 {
		return log.TraceLevel
	}

	if level >= stdslog.LevelDebug-1 && level <= stdslog.LevelDebug+1 {
		return log.DebugLevel
	}

	// Info levels
	if level >= stdslog.LevelInfo-1 && level <= stdslog.LevelInfo+1 {
		return log.InfoLevel
	}

	// Warn levels
	if level >= stdslog.LevelWarn-1 && level <= stdslog.LevelWarn+1 {
		return log.WarnLevel
	}

	// Error levels (including custom fatal/panic levels)
	if level >= stdslog.LevelError-1 && level <= stdslog.LevelError+3 {
		return log.ErrorLevel
	}

	if level >= stdslog.LevelError+4 && level <= stdslog.LevelError+7 {
		return log.FatalLevel
	}

	// Very high levels treated as panic
	return log.PanicLevel
}

// IsLevelEnabled checks if the given slog level would be enabled
// for the specified log package level
func IsLevelEnabled(currentLogLevel log.Level, slogLevel stdslog.Level) bool {
	targetLogLevel := FromSlogLevel(slogLevel)
	return targetLogLevel >= currentLogLevel
}
