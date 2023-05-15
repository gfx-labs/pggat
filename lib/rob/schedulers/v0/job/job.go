package job

import (
	"github.com/google/uuid"

	"pggat2/lib/rob"
)

type Concurrent struct {
	Source      uuid.UUID
	Constraints rob.Constraints
	Work        any
}

type Stalled struct {
	Source      uuid.UUID
	Constraints rob.Constraints
	Out         chan<- rob.Worker
}
