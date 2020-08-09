package go_async

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type T interface{}

type WorkFn func(ctx context.Context) (T, error)
type runner func() (<-chan T, <-chan error)

type ContextSource func() context.Context

type Task struct {
	// WorkFn is a cpu/io-bound workload
	parentCancelFunc context.CancelFunc
	parentContext    context.Context
	//runnerCancelFunc context.CancelFunc
	runner          runner
	resultChan      <-chan T
	errorChan       <-chan error
	cancelChan      <-chan struct{}
	state           State
	rwLock          sync.RWMutex
	result          *TaskResult
	_watcherrunning bool
}

type TaskResult struct {
	Result      T
	Error       error
	RunnerError error
}

func (t *Task) Cancel() (*TaskResult, error) {
	if t.State().IsTerminal() {
		return nil, fmt.Errorf("task state %s is terminal - cannot cancel", t.state)
	}
	// hacky - using parent context to propagate to children context so we can figure out if cancel has been called
	// without requiring the workerFn impl to be strictly cancelFn aware
	t.parentCancelFunc()
	t.setState(Cancelling)
	//should we wait/block here waiting for runner to exit?
	result, err := t._result()
	return result, err
}

func (t *Task) CancelAsync() error {
	if t.State().IsTerminal() {
		return fmt.Errorf("task state %s is terminal - cannot cancel", t.state)
	}
	t.parentCancelFunc()
	t.setState(Cancelling)
	return nil
}

func (t *Task) Result() (*TaskResult, error) {
	if t.State().IsTerminal() {
		return t.result, nil
	}
	t._run()
	t._result()
	return t.result, nil
}

func (t *Task) _result() (*TaskResult, error) {
	t._watch()
	return t.result, nil
}

func (t *Task) _watch() {
	if t._watcherrunning {
		return
	}
	var taskResult *TaskResult
	select {
	case result := <-t.resultChan:
		taskResult = &TaskResult{Result: result, Error: nil}
		t.setState(Done)
		break
	case err := <-t.errorChan:
		if IsCancelledWorker(err, *t) {
			taskResult = &TaskResult{Result: nil, Error: nil, RunnerError: err}
			t.setState(Cancelled)
		} else {
			taskResult = &TaskResult{Result: nil, Error: err}
			t.setState(Done)
		}
		break
	case <-t.cancelChan:
		// TODO does a fan-out of context.Done() make more sense?
		taskResult = &TaskResult{Result: nil, Error: nil, RunnerError: RunnerExitedFromCancellationError{}}
		t.setState(Cancelled)
		break
	}
	t.rwLock.Lock()
	defer t.rwLock.Unlock()
	t.result = taskResult
}

func (t *Task) RunAsync() error {
	if t.State().IsTerminal() {
		return fmt.Errorf("task state %s is terminal - cannot start", t.state)
	}
	t._run()
	go t._watch()
	return nil
}

func (t *Task) _run() {
	if t.State().IsTerminal() || t.State().IsRunning() {
		return
	}
	fn := func() (<-chan T, <-chan error) {
		defer t.recover()
		return t.runner()
	}
	t.resultChan, t.errorChan = fn()
	t.setState(Running)
}

func (t *Task) setState(state State) {
	if t.state.IsTerminal() {
		return
	}
	t.rwLock.Lock()
	defer t.rwLock.Unlock()
	t.state = state
}

func (t *Task) State() State {
	t.rwLock.RLock()
	defer t.rwLock.RUnlock()
	return t.state
}

func (t *Task) recover() {
	recover := recover()
	if recover == nil {
		return
	}
	err := recover.(error)
	t.result = &TaskResult{RunnerError: err}
	t.setState(InternalError)
}

func CreateTask(ctxSrc ContextSource, worker WorkFn) *Task {
	if ctxSrc == nil {
		ctxSrc = func() context.Context {
			return context.Background()
		}
	}
	parentCtx, parentCancelFn := context.WithCancel(ctxSrc())
	runnerCtx, _ := context.WithCancel(parentCtx)
	cancelChan := make(chan struct{})
	runner := func() (<-chan T, <-chan error) {
		resultChan := make(chan T)
		errorChan := make(chan error)
		go func() {
			result, err := worker(runnerCtx)
			if errors.Is(err, context.Canceled) {
				cancelChan <- struct{}{}
			}
			if err != nil {
				errorChan <- err
			} else {
				resultChan <- result
			}
		}()
		return resultChan, errorChan
	}

	task := &Task{
		rwLock: sync.RWMutex{},
		state:  Initiated,
		runner: runner,
		parentCancelFunc: parentCancelFn,
		parentContext:    parentCtx,
		cancelChan:       cancelChan}

	return task
}

func CreateAndRunAsync(ctxSrc ContextSource, worker WorkFn) *Task {
	task := CreateTask(ctxSrc, worker)
	task.RunAsync()
	return task
}
