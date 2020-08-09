package go_async_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	. "github.com/eriksywu/go-async"
)

func dummyWorkload(err error) WorkFn {
	if err == nil {
		err = RunnerExitedFromCancellationError{"WorkFn ack'ed cancellation"}
	}
	return func(ctx context.Context) (T, error) {
		timer := time.NewTimer(5 * time.Second)
		defer func() {
			timer.Stop()
		}()
		select {
		case <-ctx.Done():
			//pretend to do cleanup before cancelling
			time.Sleep(5 * time.Second)
			return nil, err
		case <-timer.C:
			return "ding", nil
		}
	}
}

func createCtxSrc() ContextSource {
	ctx := context.Background()
	ctxSrc := func() context.Context {
		return ctx
	}
	return ctxSrc
}

var ctxSrc = createCtxSrc()

func Test_TaskCompletes(t *testing.T) {
	task := CreateTask(ctxSrc, dummyWorkload(nil))
	assert.Equal(t, Initiated, task.State())

	task.RunAsync()
	assert.Equal(t, Running, task.State())

	assert.Eventually(t, func() bool { return task.State().IsTerminal() },
		10*time.Second, 1*time.Second, "did not reach terminal state")

	assert.Equal(t, Done, task.State())
	result, err := task.Result()
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "ding", result.Result.(string))

	assert.Error(t, task.RunAsync())
}

func Test_TaskReturnsResult(t *testing.T) {
	task := CreateTask(ctxSrc, dummyWorkload(nil))
	assert.Equal(t, Initiated, task.State())

	result, err := task.Result()

	assert.Equal(t, Done, task.State())
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "ding", result.Result.(string))

	assert.Error(t, task.RunAsync())
}

func Test_TaskCancels(t *testing.T) {
	task := CreateTask(ctxSrc, dummyWorkload(nil))
	assert.Equal(t, Initiated, task.State())

	task.RunAsync()
	assert.Equal(t, Running, task.State())

	task.CancelAsync()
	assert.Equal(t, Cancelling, task.State())

	assert.Eventually(t, func() bool { return task.State() == Cancelled },
		10*time.Second, 1*time.Second, "did not reach cancelled state")

	result, err := task.Result()
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, errors.Is(result.RunnerError, context.Canceled))
}

func Test_TaskCancelsWithoutContextError(t *testing.T) {
	task := CreateTask(ctxSrc, dummyWorkload(fmt.Errorf("")))
	assert.Equal(t, Initiated, task.State())

	task.RunAsync()
	assert.Equal(t, Running, task.State())

	task.CancelAsync()
	assert.Equal(t, Cancelling, task.State())

	assert.Eventually(t, func() bool { return task.State() == Cancelled },
		10*time.Second, 1*time.Second, "did not reach cancelled state")

	result, err := task.Result()
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func Test_TestExitsWithError(t *testing.T) {
	task := CreateTask(ctxSrc, func(ctx context.Context) (T, error) {
		time.Sleep(time.Second)
		return nil, fmt.Errorf("forced error")
	})
	assert.Equal(t, Initiated, task.State())

	task.RunAsync()
	assert.Equal(t, Running, task.State())

	assert.Eventually(t, func() bool { return task.State() == Done },
		10*time.Second, 1*time.Second, "did not reach done state")

	result, _ := task.Result()
	assert.NotNil(t, result)
	assert.NotNil(t, result.Error)
}
