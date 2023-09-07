package pool

type State int

const (
	StateIdle State = iota
	StateActive
	StateAwaitingServer
	StateRunningResetQuery
)
