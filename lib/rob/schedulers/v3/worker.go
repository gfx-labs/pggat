package schedulers

import (
	"time"

	"github.com/google/uuid"
)

type Worker struct {
	ID uuid.UUID

	User  *User
	Since time.Time
}
