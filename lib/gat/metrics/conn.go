package metrics

import (
	"time"

	"github.com/google/uuid"
)

type Conn struct {
	// Time this report was generated
	Time time.Time

	// Current state info

	State ConnState
	Peer  uuid.UUID
	Since time.Time

	// Period metrics

	Utilization [ConnStateCount]time.Duration

	TransactionCount int
}
