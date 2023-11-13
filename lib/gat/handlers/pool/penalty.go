package pool

import (
	"time"

	"gfx.cafe/gfx/pggat/lib/fed"
)

type Scorer interface {
	// Score calculates how much a recipe should be penalized. Lower is better
	Score(conn *fed.Conn) (score int, validity time.Duration, err error)
}
