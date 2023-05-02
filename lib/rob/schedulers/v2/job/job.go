package job

import (
	"github.com/google/uuid"

	"pggat2/lib/rob"
)

type Job struct {
	Source      uuid.UUID
	Work        any
	Constraints rob.Constraints
}
