package go_async

import (
	"context"
	"fmt"
	"sync"
)

type readiness struct {
	worker *Worker
	n      int
}

type Pool struct {
	workers    []*Worker
	ready      chan readiness
	queue      chan func()
	_started   bool
	parentCtx  context.Context
	cancelFunc context.CancelFunc
	wg         *sync.WaitGroup
}

func NewPool(size int, ctx context.Context) *Pool {
	if ctx == nil {
		ctx = context.Background()
	}
	parentCtx, cancelFunc := context.WithCancel(ctx)
	queue := make(chan func(), 1)
	ready := make(chan readiness, 1)
	wg := &sync.WaitGroup{}
	workers := make([]*Worker, size, size)
	pool := &Pool{workers: workers,
		queue:      queue,
		ready:      ready,
		wg:         wg,
		cancelFunc: cancelFunc,
		parentCtx:  parentCtx}

	for n, _ := range workers {
		workerCtx, _ := context.WithCancel(parentCtx)
		workers[n] = newWorker(n, workerCtx, func() { pool.wg.Done() })
	}

	return pool
}

func (p *Pool) Cancel() {
	p.cancelFunc()
}

func (p *Pool) CancelAsync() *Task {
	cancelWaitFn := WrapAction(func() {
		p.cancelFunc()
		p.wg.Wait()
	})
	task := CreateAndRunAsync(nil, cancelWaitFn)
	return task
}

func (p *Pool) WaitDone() {
	p.wg.Wait()
}

func (p *Pool) ChangeSize(size int) error {
	currSize := len(p.workers)
	if size < currSize {
		return fmt.Errorf("downsizing not supported")
	}
	newWorkers := make([]*Worker, size, size)
	for n, _ := range newWorkers {
		workerCtx, _ := context.WithCancel(p.parentCtx)
		worker := newWorker(n, workerCtx, func() { p.wg.Done() })
		newWorkers[n] = worker
		worker.Run(p.ready)
		p.wg.Add(1)
	}
	p.workers = append(p.workers, newWorkers...)
	return nil
}

func (p *Pool) GetQueue() chan<- func() {
	return p.queue
}

func (p *Pool) Start() {
	if p._started {
		return
	}
	go p.listen()
	p.wg.Add(len(p.workers))
	for _, w := range p.workers {
		w.Run(p.ready)
	}
	p._started = true
}

func (p *Pool) listen() {
	for work := range p.queue {
		select {
		case readyWorker := <-p.ready:
			readyWorker.worker.Request() <- work
		default:
		}
	}
}

type Worker struct {
	runner  func(ready chan<- readiness)
	request chan func()
	ctx     context.Context
	onExit  func()
}

func newWorker(n int, ctx context.Context, onExit func()) *Worker {
	worker := &Worker{ctx: ctx, onExit: onExit}
	request := make(chan func())
	runner := newRunner(request, worker, n)
	worker.runner = runner
	return worker
}

func (w *Worker) Run(ready chan<- readiness) (request chan<- func()) {
	go w.runner(ready)
	return w.request
}

func (w *Worker) Request() chan<- func() {
	return w.request
}

func newRunner(request <-chan func(), worker *Worker, n int) func(ready chan<- readiness) {
	return func(ready chan<- readiness) {
		fmt.Printf("starting runner n=%d \n", n)
		defer func() {
			recovery := recover()
			if recovery != nil {
				fmt.Printf("n=%d encountered some type of error: %v \n", n, recovery)
			}
		}()
		defer worker.onExit()

		// TODO: can simply this via tasks.WhenAny
		for {
			select {
			case <-worker.ctx.Done():
				return
			case ready <- readiness{worker: worker, n: n}:
				fmt.Printf("runner n=%d: waiting for work \n", n)
				select {
				case work := <-request:
					fmt.Printf("runner n=%d: acquired work \n", n)
					work()
				default:
					//noop?
				}
			default:
				//noop?
			}
		}
	}
}
