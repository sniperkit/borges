/*
Sniperkit-Bot
- Status: analyzed
*/

package borges

import (
	"context"
	"sync"

	"gopkg.in/src-d/go-log.v1"
	"gopkg.in/src-d/go-queue.v1"
)

// A WorkerJob is a job to be passed to the worker. It contains the Job itself
// and an acknowledger that the worker uses to signal that it finished the job.
type WorkerJob struct {
	*Job
	queueJob *queue.Job
	source   queue.Queue
}

// WorkerFunc is the function the workers will execute.
type WorkerFunc func(context.Context, log.Logger, *Job) error

// WorkerPool is a pool of workers that can process jobs.
type WorkerPool struct {
	do         WorkerFunc
	jobChannel chan *WorkerJob
	workers    []*Worker
	wg         *sync.WaitGroup
	m          *sync.Mutex
}

// NewWorkerPool creates a new empty worker pool. It takes a function to be used
// by workers to process jobs. The pool is started with no workers.
// SetWorkerCount must be called to start them.
func NewWorkerPool(f WorkerFunc) *WorkerPool {
	return &WorkerPool{
		do:         f,
		jobChannel: make(chan *WorkerJob),
		workers:    nil,
		wg:         &sync.WaitGroup{},
		m:          &sync.Mutex{},
	}
}

// Do executes a job. It blocks until a worker is assigned to process the job
// and then it returns, with the worker processing the job asynchronously.
func (wp *WorkerPool) Do(j *WorkerJob) {
	wp.jobChannel <- j
}

// SetWorkerCount changes the number of running workers. Workers will be started
// or stopped as necessary to satisfy the new worker count. It blocks until the
// all required workers are started or stopped. Each worker, if busy, will
// finish its current job before stopping.
func (wp *WorkerPool) SetWorkerCount(workers int) {
	wp.m.Lock()
	defer wp.m.Unlock()

	n := workers - len(wp.workers)
	if n > 0 {
		wp.add(n)
	} else if n < 0 {
		wp.del(-n)
	}
}

// Len returns the number of workers currently in the pool.
func (wp *WorkerPool) Len() int {
	wp.m.Lock()
	defer wp.m.Unlock()
	return len(wp.workers)
}

func (wp *WorkerPool) add(n int) {
	wp.wg.Add(n)
	for i := 0; i < n; i++ {
		logger := log.New(log.Fields{"worker": i})
		w := NewWorker(logger, wp.do, wp.jobChannel)
		go func() {
			defer wp.wg.Done()
			w.Start()
		}()
		wp.workers = append(wp.workers, w)
	}
}

func (wp *WorkerPool) del(n int) {
	prevWorkers := len(wp.workers)
	var wg sync.WaitGroup
	for i := prevWorkers - 1; i >= prevWorkers-n; i-- {
		wg.Add(1)
		w := wp.workers[i]
		wp.workers = wp.workers[:len(wp.workers)-1]
		go func() {
			w.Stop(false)
			wg.Done()
		}()
	}
	wg.Wait()
}

// Close stops all the workers in the pool and frees resources used by it
// waiting until all the current jobs finish.
func (wp *WorkerPool) Close() error {
	wp.SetWorkerCount(0)
	wp.wg.Wait()
	close(wp.jobChannel)
	return nil
}

// Stop stops all the workers in the pool and frees the resources used
// by it as well as stopping the workers and their current jobs.
func (wp *WorkerPool) Stop() error {
	for _, w := range wp.workers {
		w.Stop(true)
	}
	wp.wg.Wait()
	wp.workers = nil
	close(wp.jobChannel)
	return nil
}
