package go_async

type State int

const (
	Initiated State = iota
	Running
	Done
	Cancelling
	Cancelled
	InternalError
)

func (s State) IsTerminal() bool {
	return s == Done || s == Cancelled || s == InternalError
}

func (s State) IsRunning() bool {
	return s == Running
}
