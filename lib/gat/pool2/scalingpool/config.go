package scalingpool

import (
	"time"

	"gfx.cafe/gfx/pggat/lib/gat/pool2/recipepool"
)

type Config struct {
	// ServerIdleTimeout defines how long a server may be idle before it is disconnected.
	// 0 = disable
	ServerIdleTimeout time.Duration

	// ServerReconnectInitialTime defines how long to wait initially before attempting a server reconnect
	// 0 = disable, don't retry
	ServerReconnectInitialTime time.Duration
	// ServerReconnectMaxTime defines the max amount of time to wait before attempting a server reconnect
	// 0 = disable, back off infinitely
	ServerReconnectMaxTime time.Duration

	recipepool.Config
}
