package schedulers

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID     uuid.UUID
	Stride time.Duration

	Scheduled bool

	Worker *Worker
}
