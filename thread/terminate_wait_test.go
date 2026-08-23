package thread

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestStartAndWait_TerminateDuringWait(t *testing.T) {
	pool := NewTimeoutPool(2, 100)

	var completed int64
	// jobs slow enough that Terminate lands mid-wait
	for i := 0; i < 4; i++ {
		pool.Submit(&Job{Handler: func(_ ...interface{}) {
			time.Sleep(80 * time.Millisecond)
			atomic.AddInt64(&completed, 1)
		}})
	}

	result := make(chan bool, 1)
	go func() { result <- pool.StartAndWait(10 * time.Second) }()

	time.Sleep(20 * time.Millisecond)
	pool.Terminate()

	select {
	case ok := <-result:
		if ok {
			t.Log("jobs drained before termination signal; acceptable")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("StartAndWait did not return after Terminate")
	}
}
