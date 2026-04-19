package log

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Rotation defines the time interval for log file rotation.
type Rotation int

const (
	Hourly  Rotation = iota
	Daily
	Weekly
	Monthly
)

// RotateFile returns a Sink that writes plain text to time-rotated files.
// Files are named: {dir}/{prefix}_{timestamp}.log
func RotateFile(dir, prefix string, rotation Rotation, opts ...SinkOption) (*Sink, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("log: create dir %s: %w", dir, err)
	}

	rw := &rotateWriter{
		dir:      dir,
		prefix:   prefix,
		rotation: rotation,
	}
	if err := rw.openFile(time.Now()); err != nil {
		return nil, err
	}

	opts = append([]SinkOption{WithAsync(1024)}, opts...)
	s := newSink(NewTextEncoder(false), rw, opts...)
	s.closer = rw
	return s, nil
}

type rotateWriter struct {
	dir      string
	prefix   string
	rotation Rotation
	current  *os.File
	mu       sync.Mutex
	nextRot  time.Time
}

func (w *rotateWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if time.Now().After(w.nextRot) {
		w.rotate(time.Now())
	}
	return w.current.Write(data)
}

func (w *rotateWriter) rotate(now time.Time) {
	if w.current != nil {
		w.current.Close()
	}
	_ = w.openFile(now)
}

func (w *rotateWriter) openFile(now time.Time) error {
	ts := formatTimestamp(now, w.rotation)
	name := fmt.Sprintf("%s_%s.log", w.prefix, ts)
	path := filepath.Join(w.dir, name)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("log: open rotated file %s: %w", path, err)
	}
	w.current = f
	w.nextRot = calcNextRotation(now, w.rotation)
	return nil
}

func (w *rotateWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.current != nil {
		return w.current.Close()
	}
	return nil
}

func formatTimestamp(t time.Time, rotation Rotation) string {
	switch rotation {
	case Hourly:
		return t.Format("2006-01-02_15")
	case Daily:
		return t.Format("2006-01-02")
	case Weekly:
		y, wk := t.ISOWeek()
		return fmt.Sprintf("%d-W%02d", y, wk)
	case Monthly:
		return t.Format("2006-01")
	default:
		return t.Format("2006-01-02_15-04-05")
	}
}

func calcNextRotation(now time.Time, rotation Rotation) time.Time {
	switch rotation {
	case Hourly:
		return now.Truncate(time.Hour).Add(time.Hour)
	case Daily:
		return now.Truncate(24 * time.Hour).Add(24 * time.Hour)
	case Weekly:
		wd := now.Weekday()
		if wd == time.Sunday {
			wd = 7
		}
		weekStart := time.Date(now.Year(), now.Month(), now.Day()-int(wd-time.Monday), 0, 0, 0, 0, now.Location())
		return weekStart.Add(7 * 24 * time.Hour)
	case Monthly:
		return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	default:
		return now.AddDate(100, 0, 0)
	}
}
