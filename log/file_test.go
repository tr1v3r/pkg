package log

import (
	"fmt"
	"testing"
	"time"
)

func Test_FileName(t *testing.T) {
	logger, err := NewFileLogger("/tmp/testlog")
	if err != nil {
		t.Errorf("create new file logger fail: %s", err)
		return
	}

	expectedName := time.Now().Format("/tmp/testlog/2006-01-02T_15.log")
	if logger.FileName() != expectedName {
		t.Errorf("unexpect log file name: %s\n expect: %s", logger.FileName(), expectedName)
	}
	t.Logf("logger file name: %s", logger.FileName())

	logger, err = NewFileLogger("/tmp/testlog", FileLoggerInterval(time.Minute))
	if err != nil {
		t.Errorf("create new file logger fail: %s", err)
	}
	expectedName = time.Now().Format("/tmp/testlog/2006-01-02T_15_04.log")
	if logger.FileName() != expectedName {
		t.Errorf("unexpect log file name: %s\n expect: %s", logger.FileName(), expectedName)
	}
	t.Logf("logger file name: %s", logger.FileName())

	logger, err = NewFileLogger("/tmp/testlog", FileLoggerInterval(3*time.Hour))
	if err != nil {
		t.Errorf("create new file logger fail: %s", err)
	}
	expectedName = time.Now().Format("/tmp/testlog/2006-01-02T.log")
	if logger.FileName() != expectedName {
		t.Errorf("unexpect log file name: %s\n expect: %s", logger.FileName(), expectedName)
	}
	t.Logf("logger file name: %s", logger.FileName())
}

func TestLog(t *testing.T) {
	logger, err := NewFileLogger("/tmp/testlog",
		FileLoggerInterval(time.Minute),
		FileLoggerFormatter(func(log []byte) []byte {
			return append([]byte(time.Now().Format("2006-01-02T15:04:05")+" "), log...)
		}),
	)
	if err != nil {
		t.Errorf("create new file logger fail: %s", err)
	}

	count := 0
	for range time.Tick(time.Second) {
		n, err := logger.Write([]byte("log:" + fmt.Sprint(count) + "\n"))
		count++
		t.Logf("write log: %d %s", n, err)
	}
}
