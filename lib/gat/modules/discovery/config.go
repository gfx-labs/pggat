package discovery

import (
	"time"
)

type Config struct {
	// ReconcilePeriod is how often the module should check for changes. 0 = disable
	ReconcilePeriod time.Duration
}
