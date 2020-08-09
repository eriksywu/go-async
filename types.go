package go_async

type State string

const (
	Initiated     State = "Initiated"
	Running       State = "Running"
	Done          State = "Done"
	Cancelling    State = "Cancelling"
	Cancelled     State = "Cancelled"
	InternalError State = "InternalError"
)

func (s State) IsTerminal() bool {
	return s == Done || s == Cancelled || s == InternalError
}

func (s State) IsRunning() bool {
	return s == Running
}
