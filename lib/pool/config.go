package pool

import "time"

type BackendConfig struct {
	NewPooler func() Pooler

	IdleTimeout time.Duration

	ReconnectInitialTime time.Duration
	ReconnectMaxTime     time.Duration
}
