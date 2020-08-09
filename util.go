package go_async

import "sync"

func WrapAction(f func()) func() T {
	return func() T {
		f()
		return struct{}{}
	}
}

func WhenAny(cancelRemaining bool, tasks ...*Task) *Task {
	watcher := make(chan *Task)
	for _, task := range tasks {
		go func(t *Task) {
			t.Result()
			watcher <- t
		}(task)
	}
	completedTask := <-watcher
	if cancelRemaining {
		for _, task := range tasks {
			if task != completedTask {
				task.CancelAsync()
			}
		}
	}
	return completedTask
}

// TODO
func WhenAnyAsync(cancelRemaining bool, tasks ...*Task) *Task {
	watcher := make(chan *Task)
	for _, task := range tasks {
		go func(t *Task) {
			t.Result()
			watcher <- t
		}(task)
	}
	completedTask := <-watcher
	if cancelRemaining {
		for _, task := range tasks {
			if task != completedTask {
				task.CancelAsync()
			}
		}
	}
	return completedTask
}

func WhenAll(tasks ...*Task) {
	wg := sync.WaitGroup{}
	for _, task := range tasks {
		go func(t *Task) {
			wg.Add(1)
			t.Result()
			wg.Done()
		}(task)
	}
	wg.Wait()
}

// TODO
func WhenAllAsync(tasks ...*Task) *Task {
	wg := sync.WaitGroup{}
	for _, task := range tasks {
		go func(t *Task) {
			wg.Add(1)
			t.Result()
			wg.Done()
		}(task)
	}
	wg.Wait()
	return &Task{}
}