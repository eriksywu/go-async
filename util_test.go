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

		<-timer.C
		return int(delay), nil
	}
}

func Test_WhenAny(t *testing.T) {
	var times = []time.Duration{40, 30, 1, 20}
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
	var times = []time.Duration{40, 30, 1, 20}
	tasks := make([]*Task, 0, len(times))
	for _, delay := range times {
		tasks = append(tasks, CreateTask(nil, delayedWorkload(delay)))
	}

	completedTask := WhenAny(true, tasks...)
	assert.Equal(t, Done, completedTask.State())
	result, _ := completedTask.Result()
	assert.Equal(t, 1, result.Result.(int))
	for _, task := range tasks {
		assert.True(t, !task.State().IsRunning())
	}
}
