package job

import "github.com/google/uuid"

type Stalled struct {
	Base
	Ready chan uuid.UUID
}
