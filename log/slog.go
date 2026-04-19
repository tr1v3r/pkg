package log

import (
	"context"
	"log/slog"
)

// AsSlogHandler wraps a Sink as a slog.Handler, allowing slog output to go
// through our Sink pipeline (rotation, async, etc).
//
// Usage:
//
//	sink, _ := log.RotateFile("./logs", "app", log.Daily)
//	slog.SetDefault(slog.New(log.AsSlogHandler(sink)))
func AsSlogHandler(s *Sink) slog.Handler {
	return &sinkAsHandler{sink: s}
}

type sinkAsHandler struct {
	sink  *Sink
	group string
	attrs []slog.Attr
}

func (h *sinkAsHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *sinkAsHandler) Handle(ctx context.Context, r slog.Record) error {
	fields := make([]Field, 0, r.NumAttrs()+len(h.attrs))
	for _, a := range h.attrs {
		fields = append(fields, h.attrToField(a))
	}
	r.Attrs(func(a slog.Attr) bool {
		fields = append(fields, h.attrToField(a))
		return true
	})

	record := Record{
		Time:    r.Time,
		Level:   slogToLevel(r.Level),
		Message: r.Message,
		Fields:  fields,
		LogID:   extractLogID(ctx),
	}
	h.sink.Log(record)
	return nil
}

func (h *sinkAsHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)
	return &sinkAsHandler{sink: h.sink, group: h.group, attrs: newAttrs}
}

func (h *sinkAsHandler) WithGroup(name string) slog.Handler {
	return &sinkAsHandler{sink: h.sink, group: joinGroup(h.group, name), attrs: h.attrs}
}

func (h *sinkAsHandler) attrToField(a slog.Attr) Field {
	key := a.Key
	if h.group != "" {
		key = h.group + "." + key
	}
	return Field{Key: key, Value: a.Value.Any()}
}

func joinGroup(base, name string) string {
	if base == "" {
		return name
	}
	return base + "." + name
}

// SlogHandler wraps a slog.Handler as a Sink, allowing our log package to
// output through slog's ecosystem (sentry, loki, etc).
//
// Usage:
//
//	sentryHandler := sentryslog.NewSentryHandler(...)
//	log.Setup(log.SlogHandler(sentryHandler))
func SlogHandler(h slog.Handler, opts ...SinkOption) *Sink {
	s := &Sink{level: InfoLevel}
	for _, opt := range opts {
		opt(s)
	}
	s.logFunc = func(record Record) {
		attrs := make([]slog.Attr, len(record.Fields))
		for i, f := range record.Fields {
			attrs[i] = slog.Any(f.Key, f.Value)
		}
		r := slog.NewRecord(record.Time, levelToSlog(record.Level), record.Message, 0)
		r.AddAttrs(attrs...)
		_ = h.Handle(context.Background(), r)
	}
	return s
}

// Level conversion helpers.

func slogToLevel(l slog.Level) Level {
	switch {
	case l < slog.LevelDebug:
		return TraceLevel
	case l < slog.LevelInfo:
		return DebugLevel
	case l < slog.LevelWarn:
		return InfoLevel
	case l < slog.LevelError:
		return WarnLevel
	default:
		return ErrorLevel
	}
}

func levelToSlog(l Level) slog.Level {
	switch l {
	case TraceLevel:
		return slog.LevelDebug - 4
	case DebugLevel:
		return slog.LevelDebug
	case InfoLevel:
		return slog.LevelInfo
	case WarnLevel:
		return slog.LevelWarn
	case ErrorLevel:
		return slog.LevelError
	case FatalLevel:
		return slog.LevelError + 4
	default:
		return slog.LevelInfo
	}
}