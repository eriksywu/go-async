package go_async

import (
	"context"
	"errors"
)

type RunnerExitedFromCancellationError struct {
	Message string
}

func (e RunnerExitedFromCancellationError) Unwrap() error {
	return context.Canceled
}

func (e RunnerExitedFromCancellationError) Error() string {
	return e.Message
}

func IsCancelledWorker(err error, task Task) bool {
	if errors.Is(err, RunnerExitedFromCancellationError{}) {
		return true
	}
	if errors.Is(task.parentContext.Err(), context.Canceled) {
		return true
	}
	return false
}
