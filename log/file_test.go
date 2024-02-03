package log

import (
	"context"
	"testing"
	"time"
)

func Test_FileName(t *testing.T) {
	handler, err := NewFileHandler(TraceLevel, "/tmp/testlog")
	if err != nil {
		t.Errorf("create new file handler fail: %s", err)
		return
	}
	fileHandler := handler

	expectedName := time.Now().Format("/tmp/testlog/2006-01-02T_15.log")
	if fileHandler.FileName() != expectedName {
		t.Errorf("unexpect log file name: %s\n expect: %s", fileHandler.FileName(), expectedName)
	}
	t.Logf("handler file name: %s", fileHandler.FileName())

	handler, err = NewFileHandler(TraceLevel, "/tmp/testlog", FileHandlerInterval(time.Minute))
	if err != nil {
		t.Errorf("create new file handler fail: %s", err)
	}
	fileHandler = handler

	expectedName = time.Now().Format("/tmp/testlog/2006-01-02T_15_04.log")
	if fileHandler.FileName() != expectedName {
		t.Errorf("unexpect log file name: %s\n expect: %s", fileHandler.FileName(), expectedName)
	}
	t.Logf("handler file name: %s", fileHandler.FileName())

	handler, err = NewFileHandler(TraceLevel, "/tmp/testlog", FileHandlerInterval(3*time.Hour))
	if err != nil {
		t.Errorf("create new file handler fail: %s", err)
	}
	fileHandler = handler

	expectedName = time.Now().Format("/tmp/testlog/2006-01-02.log")
	if fileHandler.FileName() != expectedName {
		t.Errorf("unexpect log file name: %s\n expect: %s", fileHandler.FileName(), expectedName)
	}
	t.Logf("handler file name: %s", fileHandler.FileName())
}

func TestLog(t *testing.T) {
	handler, err := NewFileHandler(TraceLevel, "/tmp/testlog", FileHandlerInterval(time.Minute))
	fileHandler := handler
	if err != nil {
		t.Errorf("create new file handler fail: %s", err)
	}

	count := 0
	for range time.Tick(time.Second) {
		// n, err := fileHandler.Write([]byte("log: " + fmt.Sprint(count) + "\n"))
		count++
		// t.Logf("write log: %d %s", n, err)
		fileHandler.Output(InfoLevel, context.TODO(), "log count: %d", count)
	}
}
