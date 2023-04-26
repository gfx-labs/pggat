package gat2

import (
	"github.com/google/uuid"
)

type Source interface {
	ID() uuid.UUID

	Out() <-chan Work
}
