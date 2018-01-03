package grpool

import (
	"sync"
	"fmt"
	"time"
	"strconv"
)

// Gorouting instance which can accept client jobs
type worker struct {
	name string
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
				fmt.Println("worker",w.name," before run ",job.CallbackArgs)
				w.disp.result <- job.Run()
				fmt.Println("worker",w.name," after run ",job.CallbackArgs)
				//w.disp.wg.Done()
			case <-w.stop:
				w.stop <- struct{}{}
				return
			}
		}
	}()
}

func newWorker(name string,pool chan *worker, disp *dispatcher) *worker {
	return &worker{
		name:name,
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
	result     chan interface{}
	stop       chan struct{}
	wg         sync.WaitGroup
	CollectorCallback CollectorCallback
	CollectorStop chan struct{}
}
func (d *dispatcher) collect() {
	// wait before workers are all setup up.
	for len(d.workerPool)==0 {
		time.Sleep(time.Second)
	}
	
	for {
		select {
		case st := <-d.result:
			if len(d.workerPool)!=0{
				d.CollectorCallback(st)	
			}
		case <-d.CollectorStop:
			fmt.Println("collector stop")
			return
			
		}
	}
}
func (d *dispatcher) dispatch() {
	for {
		select {
		case job := <-d.jobQueue:
			worker := <-d.workerPool
			worker.jobChannel <- job
			//fmt.Println(worker.name," job channel:",len(d.workerPool))
		case <-d.stop:
			for i := 0; i < cap(d.workerPool); i++ {
				worker := <-d.workerPool
				fmt.Println(worker.name," stop1")
				worker.stop <-struct{}{}
				fmt.Println(worker.name," stop2")
				<-worker.stop
				fmt.Println(worker.name," stopped")
			}
			fmt.Println("dispatcher before collector stop")
			fmt.Println(cap(d.CollectorStop))
			d.CollectorStop <-struct{}{}
			fmt.Println("dispatcher after collector stop")
			return
		}
	}
}

func newDispatcher(workerPool chan *worker, jobQueue chan Job, result chan interface{}) *dispatcher {
	d := &dispatcher{
		workerPool: workerPool,
		jobQueue:   jobQueue,
		result:	result,
		stop:       make(chan struct{}),
		CollectorStop:make(chan struct{}),
	}

	for i := 0; i < cap(d.workerPool); i++ {
		worker := newWorker(strconv.Itoa(i),d.workerPool,d)
		worker.start()
	}
	go d.collect()
	go d.dispatch()
	return d
}

// Represents user request, function which should be executed in some worker.


type Job struct {
        Callback Callback
        CallbackArgs []interface{}
}
type Callback func(args []interface{}) interface{}
type CollectorCallback func(args interface{}) interface{}

func (self *Job) SetWorkerFunc(fun Callback,args...interface{} ){
        self.Callback =fun
        self.CallbackArgs =args
}

func (self *Pool) SetCollectorFunc(fun CollectorCallback){
        self.dispatcher.CollectorCallback =fun
}

func (self *Job) Run() interface{}{
        return self.Callback(self.CallbackArgs)
}


type Pool struct {
	JobQueue   chan Job
	dispatcher *dispatcher
}

// Will make pool of gorouting workers.
// numWorkers - how many workers will be created for this pool
// queueLen - how many jobs can we accept until we block
//
// Returned object contains JobQueue reference, which you can use to send job to pool.
func NewPool(numWorkers int, jobQueueLen int) *Pool {
	jobQueue := make(chan Job, jobQueueLen)
	workerPool := make(chan *worker, numWorkers)
	result := make(chan interface{}, jobQueueLen)
	
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


// How many jobs we should wait when calling WaitAll.
// It is using WaitGroup Add/Done/Wait
func (p *Pool) WaitCount(count int) {
	p.dispatcher.wg.Add(count)
}

// Will wait for all jobs to finish.
func (p *Pool) WaitAll() {
	p.dispatcher.wg.Wait()
}

// Will release resources used by pool
func (p *Pool) Release() {
	fmt.Println("release")
	p.dispatcher.stop <- struct{}{}
}
