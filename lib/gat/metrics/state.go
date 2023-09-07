package metrics

type ConnState int

const (
	ConnStateActive ConnState = iota
	ConnStateIdle
	ConnStateAwaitingServer
	ConnStateRunningResetQuery

	ConnStateCount
)

var connStateString = [ConnStateCount]string{
	ConnStateActive:            "active",
	ConnStateIdle:              "idle",
	ConnStateAwaitingServer:    "awaiting server",
	ConnStateRunningResetQuery: "running reset query",
}

func (T ConnState) String() string {
	return connStateString[T]
}
