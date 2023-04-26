package gat2

import (
	"github.com/google/uuid"
)

type Work interface {
	ID() uuid.UUID

	Source() Source
}
