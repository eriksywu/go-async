package go_async

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func dummyWorkload() worker {
	return func(ctx context.Context) (T, error) {
		timer := time.NewTimer(5 * time.Second)
		defer func() {
			timer.Stop()
		}()
		select {
		case <-ctx.Done():
			//pretend to do cleanup before cancelling
			time.Sleep(5 * time.Second)
			return nil, RunnerExitedFromCancellationError{"worker ack'ed cancellation"}
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
	task := CreateTask(ctxSrc, dummyWorkload())
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
}

func Test_TaskCancels(t *testing.T) {
	task := CreateTask(ctxSrc, dummyWorkload())
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
