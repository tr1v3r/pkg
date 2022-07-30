package thread

import (
	"sync"
	"time"
)

const (
	defaultWorkerQueueLength = 20    // 默认工作协程数量
	defaultJobQueueLength    = 10000 // 默认任务队列长度
)

// Job ...
type Job struct {
	Handler func(v ...interface{})
	Params  []interface{}
}

// TimeoutPool ...
type TimeoutPool struct {
	workerQueue chan *worker
	jobQueue    chan *Job
	jobCount    int
	jobRet      chan struct{}
	stop        chan struct{}
	terminated  chan struct{}
	lock        sync.Mutex
}

// NewTimeoutPoolWithDefaults 初始化一个带有执行超时时间的协程池，协程数量：10；任务队列长度1000
func NewTimeoutPoolWithDefaults() *TimeoutPool {
	return NewTimeoutPool(defaultWorkerQueueLength, defaultJobQueueLength)
}

// NewTimeoutPool 初始化一个带有执行超时时间的协程池，指定worker数量以及任务队列长度
func NewTimeoutPool(workerQueueLen, jobQueueLen int) *TimeoutPool {
	pool := &TimeoutPool{
		workerQueue: make(chan *worker, workerQueueLen),
		jobQueue:    make(chan *Job, jobQueueLen),
		jobRet:      make(chan struct{}, jobQueueLen),
		stop:        make(chan struct{}),
		terminated:  make(chan struct{}),
	}

	return pool
}

// Terminate 停止协程池运行，如果有正在运行中的任务会等待其运行完毕
func (p *TimeoutPool) Terminate() {
	p.stop <- struct{}{}
}

// Submit 提交一个任务到协程池
func (p *TimeoutPool) Submit(job *Job) {
	p.jobQueue <- job

	p.lock.Lock()
	p.jobCount++
	p.lock.Unlock()
}

// StartAndWaitUntilTerminated 启动并等待协程池内的运行全部运行结束 - 如果没有主动停止，如果有任务还在执行中会一直等待
// 如果返回true表示在规定时间范围内成功结束；返回false表示主动停止
// 注意：最终应该只有一个协程来调用Wait等待协程池运行结束，否则其中的计数存在竞态条件问题
func (p *TimeoutPool) StartAndWaitUntilTerminated() bool {
	// 启动协程池
	p.start()

	// 等待运行结束
	completed := 0
	for completed < p.jobCount {
		select {
		case <-p.terminated:
			return false
		default:
			select {
			case <-p.jobRet:
				completed++
			case <-p.terminated:
				return false
			}
		}
	}

	return true
}

// StartAndWait 启动并等待协程池内的运行全部运行结束
// 如果返回true表示在规定时间范围内成功结束；返回false表示运行整体超时或者主动停止
// 注意：最终应该只有一个协程来调用Wait等待协程池运行结束，否则其中的计数存在竞态条件问题
func (p *TimeoutPool) StartAndWait(timeout time.Duration) bool {
	// 启动协程池
	p.start()

	// 等待运行结束
	completed := 0
	for completed < p.jobCount {
		select {
		case <-p.terminated:
			return false
		case <-time.After(timeout):
			return false
		default:
			select {
			case <-p.jobRet:
				completed++
			case <-p.terminated:
				return false
			case <-time.After(timeout):
				return false
			}
		}
	}

	return true
}

// ~~ 内部实现

func (p *TimeoutPool) start() {
	for i := 0; i < cap(p.workerQueue); i++ {
		newWorker(p.workerQueue, p.jobRet)
	}

	go p.dispatch()
}

func (p *TimeoutPool) dispatch() {
	for {
		var job *Job
		select {
		case job = <-p.jobQueue:
			worker := <-p.workerQueue
			worker.jobChannel <- *job
		case <-p.stop:
			for i := 0; i < cap(p.workerQueue); i++ {
				worker := <-p.workerQueue
				worker.stop <- struct{}{}
				<-worker.stop
			}
			p.terminated <- struct{}{}
			return
		}
	}
}

type worker struct {
	workerQueue chan *worker
	jobChannel  chan Job
	jobRet      chan struct{}
	stop        chan struct{}
}

func (w *worker) start() {
	go func() {
		for {
			w.workerQueue <- w
			var job Job
			select {
			case job = <-w.jobChannel:
				job.Handler(job.Params...)
				w.jobRet <- struct{}{}
			case <-w.stop:
				w.stop <- struct{}{}
				return
			}
		}
	}()
}

func newWorker(workerQueue chan *worker, jobRet chan struct{}) *worker {
	worker := &worker{
		workerQueue: workerQueue,
		jobChannel:  make(chan Job),
		jobRet:      jobRet,
		stop:        make(chan struct{}),
	}

	worker.start()
	return worker
}
