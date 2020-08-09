package go_async

func WrapAction(f func()) func() T {
	return func() T {
		f()
		return struct{}{}
	}
}
