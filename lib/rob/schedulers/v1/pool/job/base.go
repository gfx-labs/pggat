package job

import (
	"time"

	"github.com/google/uuid"

	"pggat2/lib/rob"
)

type Base struct {
	Created time.Time
	ID      uuid.UUID
	Source  uuid.UUID
	Context *rob.Context
}
