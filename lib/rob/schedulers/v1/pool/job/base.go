package job

import (
	"github.com/google/uuid"

	"pggat2/lib/rob"
)

type Base struct {
	Source      uuid.UUID
	Constraints rob.Constraints
}
