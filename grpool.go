package grpool

import (
	"sync"
	"fmt"
)

// Gorouting instance which can accept client jobs
type worker struct {
	workerPool chan *worker
	jobChannel chan Job
	disp *dispatcher
	stop       chan struct{}
}

func (w *worker) start() {
	go func() {
		var job Job
		for {
			// worker free, add it to pool
			w.workerPool <- w

			select {
			case job = <-w.jobChannel:
				w.disp.result <- job()
			case <-w.stop:
				w.stop <- struct{}{}
				return
			}
		}
	}()
}

func newWorker(pool chan *worker, disp *dispatcher) *worker {
	return &worker{
		workerPool: pool,
		jobChannel: make(chan Job),
		disp:	    disp,
		stop:       make(chan struct{}),
	}
}

// Accepts jobs from clients, and waits for first free worker to deliver job
type dispatcher struct {
	workerPool chan *worker
	jobQueue   chan Job
	result     chan string
	stop       chan struct{}
}
func (d *dispatcher) collect() {
	for {
		select {
		case st := <- d.result:
			fmt.Println(st)	
		}
	}
}
func (d *dispatcher) dispatch() {
	for {
		select {
		case job := <-d.jobQueue:
			worker := <-d.workerPool
			worker.jobChannel <- job
		case  rs := <-d.result:
			fmt.Println(rs)
		case <-d.stop:
			for i := 0; i < cap(d.workerPool); i++ {
				worker := <-d.workerPool

				worker.stop <- struct{}{}
				<-worker.stop
			}

			d.stop <- struct{}{}
			return
		}
	}
}

func newDispatcher(workerPool chan *worker, jobQueue chan Job, result chan string) *dispatcher {
	d := &dispatcher{
		workerPool: workerPool,
		jobQueue:   jobQueue,
		result:	result,
		stop:       make(chan struct{}),
	}

	for i := 0; i < cap(d.workerPool); i++ {
		worker := newWorker(d.workerPool,d)
		worker.start()
	}

	go d.dispatch()
	go d.collect()
	return d
}

// Represents user request, function which should be executed in some worker.
type Job func() string

type Pool struct {
	JobQueue   chan Job
	dispatcher *dispatcher
	wg         sync.WaitGroup
}

// Will make pool of gorouting workers.
// numWorkers - how many workers will be created for this pool
// queueLen - how many jobs can we accept until we block
//
// Returned object contains JobQueue reference, which you can use to send job to pool.
func NewPool(numWorkers int, jobQueueLen int) *Pool {
	jobQueue := make(chan Job, jobQueueLen)
	workerPool := make(chan *worker, numWorkers)
	result := make(chan string, jobQueueLen)
	
	pool := &Pool{
		JobQueue:   jobQueue,
		dispatcher: newDispatcher(workerPool, jobQueue, result),
	}

	return pool
}

// In case you are using WaitAll fn, you should call this method
// every time your job is done.
//
// If you are not using WaitAll then we assume you have your own way of synchronizing.
func (p *Pool) JobDone() {
	p.wg.Done()
}

// How many jobs we should wait when calling WaitAll.
// It is using WaitGroup Add/Done/Wait
func (p *Pool) WaitCount(count int) {
	p.wg.Add(count)
}

// Will wait for all jobs to finish.
func (p *Pool) WaitAll() {
	p.wg.Wait()
}

// Will release resources used by pool
func (p *Pool) Release() {
	p.dispatcher.stop <- struct{}{}
	<-p.dispatcher.stop
}
