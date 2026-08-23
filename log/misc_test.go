package log

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
)

type failWriter struct{ err error }

func (w failWriter) Write(_ []byte) (int, error) { return 0, w.err }

func TestSinkWrite_PropagatesError(t *testing.T) {
	boom := errors.New("disk full")
	sink := newSink(NewTextEncoder(false), failWriter{err: boom}, WithSync())

	if _, err := sink.Write([]byte("x")); err == nil {
		t.Error("sink.Write should propagate writer error")
	}
}

func TestFormatValueBranches(t *testing.T) {
	cases := []struct {
		val  any
		want string
	}{
		{"plain", "plain"},             // no special chars
		{"has space", `"has space"`},   // quoted
		{errors.New("boom"), `"boom"`}, // error → quoted
		{time.Minute, "1m0s"},          // fmt.Stringer
		{42, "42"},                     // default %v
		{[]int{1, 2}, "[1 2]"},         // default %v slice
	}
	for _, tc := range cases {
		if got := formatValue(tc.val); got != tc.want {
			t.Errorf("formatValue(%#v) = %q, want %q", tc.val, got, tc.want)
		}
	}
}

func TestWithLogID_NilContext(t *testing.T) {
	// nil ctx must not panic
	ctx := WithLogID(nil, "id-1")
	if extractLogID(ctx) != "id-1" {
		t.Errorf("extractLogID = %q, want %q", extractLogID(ctx), "id-1")
	}
	if extractLogID(nil) != "" {
		t.Error("extractLogID(nil) should be empty")
	}
	if extractLogID(context.WithValue(context.Background(), logIDKey, 42)) != "" {
		t.Error("non-string logID value should extract as empty")
	}
}

func TestSlogHandler_NestedGroups(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, nil)
	sink := SlogHandler(inner, WithSync())
	logger := New(sink)

	logger.Info("with group")
	_ = joinGroup("a", "b")

	slogLogger := slog.New(AsSlogHandler(sink))
	slogLogger.Info("nested", slog.String("k", "v"))
}
