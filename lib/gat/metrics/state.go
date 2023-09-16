package metrics

type ConnState int

const (
	ConnStateIdle ConnState = iota
	ConnStateActive
	ConnStateAwaitingServer
	ConnStatePairing
	ConnStateRunningResetQuery

	ConnStateCount
)

var connStateString = [ConnStateCount]string{
	ConnStateIdle:              "idle",
	ConnStateActive:            "active",
	ConnStateAwaitingServer:    "awaiting server",
	ConnStatePairing:           "pairing",
	ConnStateRunningResetQuery: "running reset query",
}

func (T ConnState) String() string {
	return connStateString[T]
}
