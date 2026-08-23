package log

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- package-level function coverage ---

func TestPackageLevelAllLevels(t *testing.T) {
	var buf bytes.Buffer
	Setup(ConsoleTo(&buf, WithLevel(TraceLevel), WithSync()))
	defer Setup()

	ctx := WithLogID(context.Background(), "plid")

	Trace("trace msg")
	Debug("debug msg")
	Info("info msg")
	Warn("warn msg")
	Error("error msg")

	Tracef("tracef %d", 1)
	Debugf("debugf %d", 2)
	Infof("infof %d", 3)
	Warnf("warnf %d", 4)
	Errorf("errorf %d", 5)

	CtxTrace(ctx, "ctx trace")
	CtxDebug(ctx, "ctx debug")
	CtxInfo(ctx, "ctx info")
	CtxWarn(ctx, "ctx warn")
	CtxError(ctx, "ctx error")

	CtxTracef(ctx, "ctx tracef %d", 1)
	CtxDebugf(ctx, "ctx debugf %d", 2)
	CtxInfof(ctx, "ctx infof %d", 3)
	CtxWarnf(ctx, "ctx warnf %d", 4)
	CtxErrorf(ctx, "ctx errorf %d", 5)

	_ = With("k", "v")
	SetLevel(ErrorLevel)
	Info("should be filtered")
	SetLevel(TraceLevel)
	Sync()

	out := buf.String()
	for _, want := range []string{
		"trace msg", "debug msg", "info msg", "warn msg", "error msg",
		"tracef 1", "debugf 2", "infof 3", "warnf 4", "errorf 5",
		"ctx trace", "ctx debug", "ctx info", "ctx warn", "ctx error",
		"ctx tracef 1", "ctx debugf 2", "ctx infof 3", "ctx warnf 4", "ctx errorf 5",
		"[plid]",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}

	if n := strings.Count(out, "should be filtered"); n != 0 {
		t.Errorf("level filter failed, %d occurrences", n)
	}
}

func TestPackageLevelClose(t *testing.T) {
	var buf bytes.Buffer
	Setup(ConsoleTo(&buf, WithSync()))

	Info("before close")
	if err := Close(); err != nil {
		t.Errorf("Close error: %s", err)
	}
}

func TestFatalVariantsExit(t *testing.T) {
	if mode := os.Getenv("BE_FATAL_MODE"); mode != "" {
		ctx := WithLogID(context.Background(), "fatal")
		switch mode {
		case "fatalf":
			Fatalf("fatal %s", "printf")
		case "ctxfatal":
			CtxFatal(ctx, "ctx fatal")
		case "ctxfatalf":
			CtxFatalf(ctx, "ctx fatal %s", "printf")
		}
		return
	}

	for _, mode := range []string{"fatalf", "ctxfatal", "ctxfatalf"} {
		t.Run(mode, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=TestFatalVariantsExit")
			cmd.Env = append(os.Environ(), "BE_FATAL_MODE="+mode)
			err := cmd.Run()
			if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
				t.Fatalf("mode %s: expected exit code 1, got %v", mode, err)
			}
		})
	}
}

// --- field constructors ---

func TestFieldConstructors(t *testing.T) {
	err := errors.New("boom")
	cases := []struct {
		field Field
		key   string
		val   any
	}{
		{String("s", "v"), "s", "v"},
		{Int("i", 3), "i", 3},
		{Int64("i64", int64(4)), "i64", int64(4)},
		{Float64("f", 1.5), "f", 1.5},
		{Bool("b", true), "b", true},
		{Err(err), "err", err},
		{Duration("d", time.Second), "d", time.Second},
		{Any("a", struct{}{}), "a", struct{}{}},
	}
	for _, tc := range cases {
		if tc.field.Key != tc.key || tc.field.Value != tc.val {
			t.Errorf("field = {%s %v}, want {%s %v}", tc.field.Key, tc.field.Value, tc.key, tc.val)
		}
	}
}

// --- rotation internals ---

func TestFormatTimestamp(t *testing.T) {
	ts := time.Date(2024, 6, 2, 10, 20, 30, 0, time.UTC) // Sunday, ISO week 22
	cases := []struct {
		rotation Rotation
		want     string
	}{
		{Hourly, "2024-06-02_10"},
		{Daily, "2024-06-02"},
		{Weekly, "2024-W22"},
		{Monthly, "2024-06"},
		{Rotation(99), "2024-06-02_10-20-30"},
	}
	for _, tc := range cases {
		if got := formatTimestamp(ts, tc.rotation); got != tc.want {
			t.Errorf("formatTimestamp(%v) = %q, want %q", tc.rotation, got, tc.want)
		}
	}
}

func TestCalcNextRotation(t *testing.T) {
	utc := func(s string) time.Time { ts, _ := time.Parse(time.RFC3339, s); return ts }

	cases := []struct {
		name     string
		now      time.Time
		rotation Rotation
		want     time.Time
	}{
		{"hourly", utc("2024-06-01T10:30:00Z"), Hourly, utc("2024-06-01T11:00:00Z")},
		{"daily", utc("2024-06-01T10:30:00Z"), Daily, utc("2024-06-02T00:00:00Z")},
		{"weekly-sunday", utc("2024-06-02T10:00:00Z"), Weekly, utc("2024-06-03T00:00:00Z")},
		{"weekly-midweek", utc("2024-05-29T10:00:00Z"), Weekly, utc("2024-06-03T00:00:00Z")},
		{"monthly", utc("2024-06-15T10:00:00Z"), Monthly, utc("2024-07-01T00:00:00Z")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := calcNextRotation(tc.now, tc.rotation); !got.Equal(tc.want) {
				t.Errorf("calcNextRotation = %v, want %v", got, tc.want)
			}
		})
	}

	// unknown rotation pushes far into the future
	next := calcNextRotation(time.Now(), Rotation(99))
	if !next.After(time.Now().AddDate(99, 0, 0)) {
		t.Errorf("unknown rotation next = %v, want ~100 years out", next)
	}
}

func TestRotateWriterRotatesAcrossBoundary(t *testing.T) {
	dir := t.TempDir()

	rw := &rotateWriter{dir: dir, prefix: "app", rotation: Daily}
	if err := rw.openFile(time.Now()); err != nil {
		t.Fatalf("openFile fail: %s", err)
	}

	first := rw.current

	// force the rotation deadline into the past, then write
	rw.nextRot = time.Now().Add(-time.Second)
	if _, err := rw.Write([]byte("after boundary")); err != nil {
		t.Fatalf("write fail: %s", err)
	}

	if rw.current == first {
		t.Error("writer should have rotated to a new file")
	}
	if !rw.nextRot.After(time.Now()) {
		t.Error("nextRot should be in the future after rotating")
	}
	if err := rw.Close(); err != nil {
		t.Errorf("close fail: %s", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		t.Errorf("expected rotated files in dir: %v %v", entries, err)
	}
}

func TestSizeRotateFindLastSeq(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"app_001.log", "app_003.log", "app_ignore.log", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "app_009.log"), 0o755); err != nil {
		t.Fatal(err) // directory with matching name must be skipped
	}

	w := &sizeWriter{dir: dir, prefix: "app"}
	if got := w.findLastSeq(); got != 3 {
		t.Errorf("findLastSeq = %d, want 3", got)
	}

	// empty dir → 0
	if got := (&sizeWriter{dir: t.TempDir(), prefix: "app"}).findLastSeq(); got != 0 {
		t.Errorf("findLastSeq empty dir = %d, want 0", got)
	}

	// missing dir → 0
	if got := (&sizeWriter{dir: filepath.Join(dir, "nope"), prefix: "app"}).findLastSeq(); got != 0 {
		t.Errorf("findLastSeq missing dir = %d, want 0", got)
	}
}

// --- sink io.Writer path & file errors ---

func TestSinkWriteMethod(t *testing.T) {
	var buf bytes.Buffer
	sink := newSink(NewTextEncoder(false), &buf, WithSync())

	n, err := sink.Write([]byte("raw bytes"))
	if err != nil {
		t.Fatalf("sink.Write fail: %s", err)
	}
	if n != len("raw bytes") {
		t.Errorf("sink.Write n = %d, want %d", n, len("raw bytes"))
	}
	if !strings.Contains(buf.String(), "raw bytes") {
		t.Errorf("buffer = %q", buf.String())
	}
}

func TestFileUnopenable(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "missing-dir", "app.log")
	if s, err := File(bad); err == nil {
		if s != nil {
			_ = s.Close()
		}
		t.Error("File on unopenable path should fail")
	}
}

func TestRotateFileBadDir(t *testing.T) {
	// dir path occupied by a regular file → MkdirAll fails
	file := filepath.Join(t.TempDir(), "occupied")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := RotateFile(filepath.Join(file, "sub"), "app", Daily); err == nil {
		t.Error("RotateFile under a file path should fail")
	}
}

// --- slog level mapping ---

func TestSlogLevelConversions(t *testing.T) {
	if got := slogToLevel(slog.LevelDebug - 4); got != TraceLevel {
		t.Errorf("slogToLevel(Debug-4) = %v, want Trace", got)
	}
	if got := slogToLevel(slog.LevelDebug); got != DebugLevel {
		t.Errorf("slogToLevel(Debug) = %v", got)
	}
	if got := slogToLevel(slog.LevelInfo); got != InfoLevel {
		t.Errorf("slogToLevel(Info) = %v", got)
	}
	if got := slogToLevel(slog.LevelWarn); got != WarnLevel {
		t.Errorf("slogToLevel(Warn) = %v", got)
	}
	if got := slogToLevel(slog.LevelError); got != ErrorLevel {
		t.Errorf("slogToLevel(Error) = %v", got)
	}
	if got := slogToLevel(slog.Level(12)); got != ErrorLevel { // at/above Error maps to Error
		t.Errorf("slogToLevel(12) = %v, want Error", got)
	}

	if got := levelToSlog(TraceLevel); got != slog.LevelDebug-4 {
		t.Errorf("levelToSlog(Trace) = %v", got)
	}
	if got := levelToSlog(DebugLevel); got != slog.LevelDebug {
		t.Errorf("levelToSlog(Debug) = %v", got)
	}
	if got := levelToSlog(InfoLevel); got != slog.LevelInfo {
		t.Errorf("levelToSlog(Info) = %v", got)
	}
	if got := levelToSlog(WarnLevel); got != slog.LevelWarn {
		t.Errorf("levelToSlog(Warn) = %v", got)
	}
	if got := levelToSlog(ErrorLevel); got != slog.LevelError {
		t.Errorf("levelToSlog(Error) = %v", got)
	}
	if got := levelToSlog(FatalLevel); got != slog.LevelError+4 {
		t.Errorf("levelToSlog(Fatal) = %v", got)
	}
	if got := levelToSlog(Level(99)); got != slog.LevelInfo {
		t.Errorf("levelToSlog(99) = %v, want Info default", got)
	}
}

// --- JSON encoder with caller/logID round trip ---

func TestJSONEncoderAllFields(t *testing.T) {
	var buf bytes.Buffer
	sink := newSink(NewJSONEncoder(), &buf, WithSync())
	logger := New(sink)
	logger.Info("json all", String("s", "v"), Int("n", 1))

	out := buf.String()
	for _, want := range []string{"\"msg\":\"json all\"", "\"s\":\"v\"", "\"n\":1"} {
		if !strings.Contains(out, want) {
			t.Errorf("json output missing %s: %s", want, out)
		}
	}
	if !strings.HasSuffix(out, "}\n") {
		t.Errorf("json output should end with newline: %q", out)
	}
}

// TestFatalfFlushesAsyncSink guards the fix for Fatal exiting before async
// sinks drained: a subprocess writes a fatal record to an async file sink,
// and the parent asserts the record made it to disk.
func TestFatalfFlushesAsyncSink(t *testing.T) {
	if os.Getenv("BE_FATAL_FILE") != "" {
		path := os.Getenv("BE_FATAL_FILE")
		sink, err := File(path) // async by default (WithAsync(1024))
		if err != nil {
			os.Exit(2)
		}
		Setup(sink)
		Fatalf("flush me before exit")
		return
	}

	path := filepath.Join(t.TempDir(), "fatal.log")
	cmd := exec.Command(os.Args[0], "-test.run=TestFatalfFlushesAsyncSink")
	cmd.Env = append(os.Environ(), "BE_FATAL_FILE="+path)
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got %v", err)
	}

	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("fatal log file missing: %s", readErr)
	}
	if !strings.Contains(string(data), "flush me before exit") {
		t.Errorf("fatal record not flushed to async sink, file = %q", data)
	}
}
