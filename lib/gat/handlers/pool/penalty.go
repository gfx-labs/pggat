package pool

import (
	"time"

	"gfx.cafe/gfx/pggat/lib/fed"
)

type Critic interface {
	// Taste calculates how much conn should be penalized. Lower is better
	Taste(conn fed.Conn) (score int, validity time.Duration, err error)
}
