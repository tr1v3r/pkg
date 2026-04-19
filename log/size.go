package log

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// SizeRotateFile returns a Sink that rotates files when they exceed maxSize bytes.
// Files are named: {dir}/{prefix}_001.log, {dir}/{prefix}_002.log, ...
func SizeRotateFile(dir, prefix string, maxSize int64, opts ...SinkOption) (*Sink, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("log: create dir %s: %w", dir, err)
	}

	sw := &sizeWriter{
		dir:     dir,
		prefix:  prefix,
		maxSize: maxSize,
	}
	sw.seq = sw.findLastSeq() + 1
	if err := sw.openFile(); err != nil {
		return nil, err
	}

	opts = append([]SinkOption{WithAsync(1024)}, opts...)
	s := newSink(NewTextEncoder(false), sw, opts...)
	s.closer = sw
	return s, nil
}

type sizeWriter struct {
	dir     string
	prefix  string
	maxSize int64
	current *os.File
	written int64
	seq     int
	mu      sync.Mutex
}

func (w *sizeWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.written+int64(len(data)) > w.maxSize && w.written > 0 {
		w.rotate()
	}

	n, err := w.current.Write(data)
	w.written += int64(n)
	return n, err
}

func (w *sizeWriter) rotate() {
	w.current.Close()
	w.seq++
	_ = w.openFile()
}

func (w *sizeWriter) openFile() error {
	name := fmt.Sprintf("%s_%03d.log", w.prefix, w.seq)
	path := filepath.Join(w.dir, name)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("log: open size-rotated file %s: %w", path, err)
	}
	w.current = f
	w.written = 0
	return nil
}

func (w *sizeWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.current != nil {
		return w.current.Close()
	}
	return nil
}

func (w *sizeWriter) findLastSeq() int {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return 0
	}
	prefix := w.prefix + "_"
	maxSeq := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), prefix) {
			continue
		}
		base := strings.TrimSuffix(e.Name(), ".log")
		seqStr := strings.TrimPrefix(base, prefix)
		seq, err := strconv.Atoi(seqStr)
		if err != nil {
			continue
		}
		if seq > maxSeq {
			maxSeq = seq
		}
	}
	return maxSeq
}
