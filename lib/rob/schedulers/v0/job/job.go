package job

import (
	"github.com/google/uuid"
	"pggat2/lib/rob"
)

type Job struct {
	Source      uuid.UUID
	Constraints rob.Constraints
	Work        any
}
