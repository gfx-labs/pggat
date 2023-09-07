package pool

type State int

const (
	StateActive State = iota
	StateIdle
	StateAwaitingServer
	StateRunningResetQuery
)
