package job

import (
	"github.com/google/uuid"
)

type Stalled struct {
	Concurrent
	Ready chan<- uuid.UUID
}
