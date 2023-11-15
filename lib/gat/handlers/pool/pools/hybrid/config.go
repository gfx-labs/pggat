package hybrid

import (
	"time"

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/critics/latency"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/poolers/rob"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/spool"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type Config struct {
	ServerIdleTimeout caddy.Duration `json:"server_idle_timeout,omitempty"`

	ServerReconnectInitialTime caddy.Duration `json:"server_reconnect_initial_time,omitempty"`
	ServerReconnectMaxTime     caddy.Duration `json:"server_reconnect_max_time,omitempty"`

	TrackedParameters []strutil.CIString `json:"tracked_parameters,omitempty"`

	Logger *zap.Logger `json:"-"`
}

func (T Config) Spool() spool.Config {
	return spool.Config{
		PoolerFactory:        new(rob.Factory),
		UsePS:                true,
		UseEQP:               true,
		IdleTimeout:          time.Duration(T.ServerIdleTimeout),
		ReconnectInitialTime: time.Duration(T.ServerReconnectInitialTime),
		ReconnectMaxTime:     time.Duration(T.ServerReconnectMaxTime),

		Critics: []pool.Critic{
			&latency.Critic{
				Threshold: caddy.Duration(200 * time.Millisecond),
			},
		},

		Logger: T.Logger,
	}
}
