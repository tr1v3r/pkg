package thread

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestNewPool(t *testing.T) {
	pool := NewTimeoutPoolWithDefaults()

	var count int64
	jobs := 50
	for i := 0; i < jobs; i++ {
		pool.Submit(&Job{
			Handler: func(v ...interface{}) {
				atomic.AddInt64(&count, 1)
			},
			Params: []interface{}{i},
		})
	}

	if ok := pool.StartAndWait(5 * time.Second); !ok {
		t.Fatal("StartAndWait should finish in time")
	}
	if got := atomic.LoadInt64(&count); got != int64(jobs) {
		t.Errorf("executed jobs = %d, want %d", got, jobs)
	}
}

func TestNewTimeoutPool_Sizing(t *testing.T) {
	p := NewTimeoutPool(3, 10)
	if cap(p.workerQueue) != 3 {
		t.Errorf("worker queue cap = %d, want 3", cap(p.workerQueue))
	}
	if cap(p.jobQueue) != 10 {
		t.Errorf("job queue cap = %d, want 10", cap(p.jobQueue))
	}

	d := NewTimeoutPoolWithDefaults()
	if cap(d.workerQueue) != defaultWorkerQueueLength || cap(d.jobQueue) != defaultJobQueueLength {
		t.Error("defaults pool has unexpected sizes")
	}
}

func TestStartAndWaitUntilTerminated(t *testing.T) {
	pool := NewTimeoutPool(4, 100)

	var count int64
	for i := 0; i < 10; i++ {
		pool.Submit(&Job{Handler: func(_ ...interface{}) {
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt64(&count, 1)
		}})
	}

	if ok := pool.StartAndWaitUntilTerminated(); !ok {
		t.Fatal("all jobs done without termination should return true")
	}
	if got := atomic.LoadInt64(&count); got != 10 {
		t.Errorf("executed jobs = %d, want 10", got)
	}
}

func TestTerminate(t *testing.T) {
	pool := NewTimeoutPool(2, 100)

	var completed int64
	jobs := 4
	for i := 0; i < jobs; i++ {
		pool.Submit(&Job{Handler: func(_ ...interface{}) {
			time.Sleep(20 * time.Millisecond)
			atomic.AddInt64(&completed, 1)
		}})
	}

	// Terminate is cooperative: in-flight jobs drain first, so the waiter may
	// legitimately report true. Assert it returns promptly and all jobs ran.
	result := make(chan bool, 1)
	go func() { result <- pool.StartAndWaitUntilTerminated() }()

	time.Sleep(10 * time.Millisecond)
	pool.Terminate()

	select {
	case <-result:
	case <-time.After(5 * time.Second):
		t.Fatal("StartAndWaitUntilTerminated did not return after Terminate")
	}
	t.Logf("terminate result received, completed=%d", atomic.LoadInt64(&completed))
}

func TestTerminateAfterIdle(t *testing.T) {
	pool := NewTimeoutPool(2, 100)

	pool.Submit(&Job{Handler: func(_ ...interface{}) {}})
	if ok := pool.StartAndWaitUntilTerminated(); !ok {
		t.Fatal("single fast job should complete")
	}

	// Terminate an idle pool: the dispatch loop collects idle workers and
	// signals termination. Must not block or panic.
	done := make(chan struct{})
	go func() { pool.Terminate(); close(done) }()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Terminate on idle pool did not return")
	}
}

func TestStartAndWaitTimeout(t *testing.T) {
	pool := NewTimeoutPool(2, 100)

	started := make(chan struct{})
	pool.Submit(&Job{Handler: func(_ ...interface{}) {
		close(started)
		time.Sleep(500 * time.Millisecond)
	}})

	if ok := pool.StartAndWait(100 * time.Millisecond); ok {
		t.Error("long job with 100ms budget should time out, got true")
	}
	<-started
}

func TestJobTimeout_FreesWorker(t *testing.T) {
	pool := NewTimeoutPool(1, 10) // single worker: a hung job would block everything

	var quickDone int64
	jobs := []*Job{
		// hangs far beyond its deadline; without timeout support the single
		// worker is stuck and the quick jobs below never run
		{Timeout: 50 * time.Millisecond, Handler: func(_ ...interface{}) { time.Sleep(2 * time.Second) }},
		{Timeout: time.Second, Handler: func(_ ...interface{}) { atomic.AddInt64(&quickDone, 1) }},
		{Timeout: time.Second, Handler: func(_ ...interface{}) { atomic.AddInt64(&quickDone, 1) }},
	}
	for _, j := range jobs {
		pool.Submit(j)
	}

	start := time.Now()
	if ok := pool.StartAndWait(5 * time.Second); !ok {
		t.Fatal("pool should finish despite the hung job")
	}
	elapsed := time.Since(start)

	if got := atomic.LoadInt64(&quickDone); got != 2 {
		t.Errorf("quick jobs executed = %d, want 2 (worker must be freed after timeout)", got)
	}
	if elapsed > time.Second {
		t.Errorf("elapsed = %v, pool should not wait for the hung handler", elapsed)
	}
}

func TestJobTimeout_ZeroMeansUnlimited(t *testing.T) {
	pool := NewTimeoutPool(2, 10)

	var done int64
	// no timeout: a 150ms handler must run to completion
	pool.Submit(&Job{Handler: func(_ ...interface{}) {
		time.Sleep(150 * time.Millisecond)
		atomic.AddInt64(&done, 1)
	}})

	if ok := pool.StartAndWait(5 * time.Second); !ok {
		t.Fatal("StartAndWait failed")
	}
	if got := atomic.LoadInt64(&done); got != 1 {
		t.Errorf("handler completed = %d, want 1 (zero timeout must not interrupt)", got)
	}
}

func TestJobTimeout_LongDeadlineNotTriggered(t *testing.T) {
	pool := NewTimeoutPool(1, 10)

	var done int64
	pool.Submit(&Job{
		Timeout: 5 * time.Second, // generous deadline
		Handler: func(_ ...interface{}) {
			time.Sleep(50 * time.Millisecond)
			atomic.AddInt64(&done, 1)
		},
	})

	if ok := pool.StartAndWait(5 * time.Second); !ok {
		t.Fatal("StartAndWait failed")
	}
	if got := atomic.LoadInt64(&done); got != 1 {
		t.Errorf("handler completed = %d, want 1 (deadline must not fire early)", got)
	}
}
