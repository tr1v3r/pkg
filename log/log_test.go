package log

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// --- Level tests ---

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{TraceLevel, "TRACE"},
		{DebugLevel, "DEBUG"},
		{InfoLevel, "INFO"},
		{WarnLevel, "WARN"},
		{ErrorLevel, "ERROR"},
		{FatalLevel, "FATAL"},
	}
	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("Level(%d).String() = %s, want %s", tt.level, got, tt.expected)
		}
	}
}

// --- toFields tests ---

func TestToFields(t *testing.T) {
	fields := toFields("user", "alice", "port", 8080)
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(fields))
	}
	if fields[0].Key != "user" || fields[0].Value != "alice" {
		t.Errorf("field[0] = %v, want {user, alice}", fields[0])
	}
	if fields[1].Key != "port" || fields[1].Value != 8080 {
		t.Errorf("field[1] = %v, want {port, 8080}", fields[1])
	}
}

func TestToFields_Empty(t *testing.T) {
	if fields := toFields(); fields != nil {
		t.Errorf("expected nil for empty args, got %v", fields)
	}
}

func TestToFields_OddArgs(t *testing.T) {
	fields := toFields("user", "alice", "orphan")
	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}
	if fields[0].Key != "user" {
		t.Errorf("field[0].Key = %s, want user", fields[0].Key)
	}
}

// --- Context tests ---

func TestWithLogID(t *testing.T) {
	ctx := WithLogID(context.Background(), "req-123")
	if id := extractLogID(ctx); id != "req-123" {
		t.Errorf("extractLogID = %s, want req-123", id)
	}
}

func TestNewLogID(t *testing.T) {
	id := NewLogID()
	if id == "" {
		t.Error("NewLogID returned empty string")
	}
	if len(id) != 36 { // UUID v4 format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
		t.Errorf("NewLogID length = %d, want 36", len(id))
	}
}

// --- TextEncoder tests ---

func TestTextEncoder(t *testing.T) {
	enc := NewTextEncoder(false)
	record := Record{
		Time:    parseTime("2024-01-01T12:00:00Z"),
		Level:   InfoLevel,
		Message: "request received",
		Fields:  []Field{{Key: "method", Value: "GET"}},
	}
	out := string(enc.Encode(record))
	if !strings.Contains(out, "[INFO]") {
		t.Errorf("expected [INFO] in output, got: %s", out)
	}
	if !strings.Contains(out, "request received") {
		t.Errorf("expected message in output, got: %s", out)
	}
	if !strings.Contains(out, "method=GET") {
		t.Errorf("expected field in output, got: %s", out)
	}
}

func TestTextEncoder_LogID(t *testing.T) {
	enc := NewTextEncoder(false)
	record := Record{
		Time:    parseTime("2024-01-01T12:00:00Z"),
		Level:   InfoLevel,
		Message: "test",
		LogID:   "req-abc",
	}
	out := string(enc.Encode(record))
	if !strings.Contains(out, "[req-abc]") {
		t.Errorf("expected [req-abc] in output, got: %s", out)
	}
}

// --- JSONEncoder tests ---

func TestJSONEncoder(t *testing.T) {
	enc := NewJSONEncoder()
	record := Record{
		Time:    parseTime("2024-01-01T12:00:00Z"),
		Level:   InfoLevel,
		Message: "hello",
		Fields:  []Field{{Key: "user", Value: "alice"}},
	}
	out := string(enc.Encode(record))
	if !strings.Contains(out, `"level":"INFO"`) {
		t.Errorf("expected level in JSON, got: %s", out)
	}
	if !strings.Contains(out, `"msg":"hello"`) {
		t.Errorf("expected message in JSON, got: %s", out)
	}
	if !strings.Contains(out, `"user":"alice"`) {
		t.Errorf("expected field in JSON, got: %s", out)
	}
}

// --- Sink + Logger integration tests ---

func newTestSink() (*Sink, *bytes.Buffer) {
	var buf bytes.Buffer
	return newSink(NewTextEncoder(false), &buf, WithLevel(TraceLevel)), &buf
}

func TestLogger_Structured(t *testing.T) {
	sink, buf := newTestSink()
	logger := New(sink)

	logger.Info("hello", "user", "alice", "port", 8080)

	out := buf.String()
	if !strings.Contains(out, "hello") {
		t.Errorf("expected message, got: %s", out)
	}
	if !strings.Contains(out, "user=alice") {
		t.Errorf("expected field, got: %s", out)
	}
	if !strings.Contains(out, "port=8080") {
		t.Errorf("expected field, got: %s", out)
	}
}

func TestLogger_Printf(t *testing.T) {
	sink, buf := newTestSink()
	logger := New(sink)

	logger.Infof("server started on :%d", 8080)

	out := buf.String()
	if !strings.Contains(out, "server started on :8080") {
		t.Errorf("expected formatted message, got: %s", out)
	}
}

func TestLogger_CtxStructured(t *testing.T) {
	sink, buf := newTestSink()
	logger := New(sink)

	ctx := WithLogID(context.Background(), "req-456")
	logger.CtxInfo(ctx, "got response", "bytes", 1024)

	out := buf.String()
	if !strings.Contains(out, "got response") {
		t.Errorf("expected message, got: %s", out)
	}
	if !strings.Contains(out, "bytes=1024") {
		t.Errorf("expected structured field, got: %s", out)
	}
	if !strings.Contains(out, "[req-456]") {
		t.Errorf("expected logID, got: %s", out)
	}
}

func TestLogger_CtxPrintf(t *testing.T) {
	sink, buf := newTestSink()
	logger := New(sink)

	ctx := WithLogID(context.Background(), "req-789")
	logger.CtxInfof(ctx, "got response %d bytes", 1024)

	out := buf.String()
	if !strings.Contains(out, "got response 1024 bytes") {
		t.Errorf("expected formatted message, got: %s", out)
	}
	if !strings.Contains(out, "[req-789]") {
		t.Errorf("expected logID, got: %s", out)
	}
}

func TestLogger_With(t *testing.T) {
	sink, buf := newTestSink()
	logger := New(sink)

	child := logger.With("service", "api")
	child.Info("request handled", "method", "GET")

	out := buf.String()
	if !strings.Contains(out, "service=api") {
		t.Errorf("expected preset field, got: %s", out)
	}
	if !strings.Contains(out, "method=GET") {
		t.Errorf("expected per-call field, got: %s", out)
	}
}

func TestLogger_LevelFilter(t *testing.T) {
	sink, buf := newTestSink()
	sink.SetLevel(WarnLevel)
	logger := New(sink)

	logger.Info("should be filtered")
	logger.Warn("should appear")

	out := buf.String()
	if strings.Contains(out, "should be filtered") {
		t.Error("info should be filtered at warn level")
	}
	if !strings.Contains(out, "should appear") {
		t.Error("warn should pass through")
	}
}

// --- Global function tests ---

func TestGlobalFunctions(t *testing.T) {
	sink, buf := newTestSink()
	Setup(sink)

	Info("global test", "key", "value")
	if !strings.Contains(buf.String(), "global test") {
		t.Errorf("global Info failed, got: %s", buf.String())
	}

	buf.Reset()
	ctx := WithLogID(context.Background(), "global-123")
	CtxDebugf(ctx, "ctx test %s", "arg")
	if !strings.Contains(buf.String(), "ctx test arg") {
		t.Errorf("global CtxDebugf failed, got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "[global-123]") {
		t.Errorf("global CtxDebugf missing logID, got: %s", buf.String())
	}

	buf.Reset()
	Infof("printf %d", 42)
	if !strings.Contains(buf.String(), "printf 42") {
		t.Errorf("global Infof failed, got: %s", buf.String())
	}
}

// --- File sink test ---

func TestFileSink(t *testing.T) {
	tmpDir := t.TempDir()
	sink, err := File(tmpDir + "/test.log", WithLevel(DebugLevel))
	if err != nil {
		t.Fatal(err)
	}
	defer sink.Close()

	logger := New(sink)
	logger.Info("file test", "key", "value")
	sink.Sync()

	data, err := os.ReadFile(tmpDir + "/test.log")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "file test") {
		t.Errorf("expected message in file, got: %s", string(data))
	}
	if !strings.Contains(string(data), "key=value") {
		t.Errorf("expected field in file, got: %s", string(data))
	}
}

// --- RotateFile sink test ---

func TestRotateFileSink(t *testing.T) {
	tmpDir := t.TempDir()
	sink, err := RotateFile(tmpDir, "app", Hourly, WithLevel(InfoLevel))
	if err != nil {
		t.Fatal(err)
	}
	defer sink.Close()

	logger := New(sink)
	logger.Info("rotate test")
	sink.Sync()

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Error("expected at least one log file")
	}
}

// --- SizeRotateFile sink test ---

func TestSizeRotateFileSink(t *testing.T) {
	tmpDir := t.TempDir()
	// 100 bytes max to trigger rotation quickly
	sink, err := SizeRotateFile(tmpDir, "app", 100, WithLevel(TraceLevel))
	if err != nil {
		t.Fatal(err)
	}
	defer sink.Close()

	logger := New(sink)
	for i := 0; i < 20; i++ {
		logger.Info("test message that should fill up the file", "iter", i)
	}
	sink.Sync()

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 2 {
		t.Errorf("expected rotation (>= 2 files), got %d files", len(entries))
	}
}

// --- JSON sink test ---

func TestJSONSink(t *testing.T) {
	var buf bytes.Buffer
	sink := newSink(NewJSONEncoder(), &buf, WithLevel(InfoLevel))

	logger := New(sink)
	logger.Info("json test", "user", "bob")

	out := buf.String()
	if !strings.Contains(out, `"msg":"json test"`) {
		t.Errorf("expected JSON message, got: %s", out)
	}
	if !strings.Contains(out, `"user":"bob"`) {
		t.Errorf("expected JSON field, got: %s", out)
	}
}

// --- Fatal test (subprocess) ---

func TestFatalExits(t *testing.T) {
	if os.Getenv("BE_FATAL") == "1" {
		Fatal("fatal test")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestFatalExits")
	cmd.Env = append(os.Environ(), "BE_FATAL=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got %v", err)
	}
}

// --- Multi-sink test ---

func TestMultiSink(t *testing.T) {
	sink1, buf1 := newTestSink()
	sink2, buf2 := newTestSink()

	logger := New(sink1, sink2)
	logger.Info("multi test")

	if !strings.Contains(buf1.String(), "multi test") {
		t.Error("sink1 should receive message")
	}
	if !strings.Contains(buf2.String(), "multi test") {
		t.Error("sink2 should receive message")
	}
}

// --- Benchmark ---

func BenchmarkTextEncode(b *testing.B) {
	enc := NewTextEncoder(false)
	record := Record{
		Level:   InfoLevel,
		Message: "benchmark message",
		Fields:  []Field{{Key: "key1", Value: "val1"}, {Key: "key2", Value: 42}},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enc.Encode(record)
	}
}

func BenchmarkLoggerWrite(b *testing.B) {
	sink := newSink(NewTextEncoder(false), ioDiscarder{}, WithLevel(InfoLevel))
	logger := New(sink)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark", "iter", i)
	}
}

// helpers

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

type ioDiscarder struct{}

func (ioDiscarder) Write(p []byte) (int, error) { return len(p), nil }
