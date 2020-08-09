package go_async_test

import (
	"context"
	"fmt"
	. "github.com/eriksywu/go-async"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func delayedWorkload(delay time.Duration) WorkFn {
	return func(ctx context.Context) (T, error) {
		timer := time.NewTimer(time.Second * delay)
		fmt.Printf("gonna sleep for delay=%d \n", delay)
		select {
		case <-timer.C:
			return int(delay), nil
		case <-ctx.Done():
			return nil, RunnerExitedFromCancellationError{}
		}
	}
}

func Test_WhenAny(t *testing.T) {
	var times = []time.Duration{4, 3, 1}
	tasks := make([]*Task, 0, len(times))
	for _, delay := range times {
		tasks = append(tasks, CreateTask(nil, delayedWorkload(delay)))
	}

	completedTask := WhenAny(false, tasks...)
	assert.Equal(t, Done, completedTask.State())
	result, _ := completedTask.Result()
	assert.Equal(t, 1, result.Result.(int))
}

func Test_WhenAnyCancelAfter(t *testing.T) {
	var times = []time.Duration{4, 3, 1}
	tasks := make([]*Task, 0, len(times))
	for _, delay := range times {
		tasks = append(tasks, CreateTask(nil, delayedWorkload(delay)))
	}

	completedTask := WhenAny(true, tasks...)
	assert.Equal(t, Done, completedTask.State())
	result, _ := completedTask.Result()
	// this could be brittle
	assert.Equal(t, 1, result.Result.(int))
	for _, task := range tasks {
		assert.True(t, !task.State().IsRunning())
		if !task.State().IsTerminal() {
			assert.Eventually(t, func() bool { return task.State() == Cancelled}, 10*time.Second, time.Second)
		}
	}
}
