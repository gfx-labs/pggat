package kitchen

import "time"

type Rating struct {
	Expiration time.Time
	Score      int
}
