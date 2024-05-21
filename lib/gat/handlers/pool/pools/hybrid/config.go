package hybrid

import (
	"encoding/json"
	"time"

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/poolers/rob"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/spool"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type Config struct {
	ClientAcquireTimeout caddy.Duration `json:"client_acquire_timeout,omitempty"`

	ServerIdleTimeout caddy.Duration `json:"server_idle_timeout,omitempty"`

	ServerReconnectInitialTime caddy.Duration `json:"server_reconnect_initial_time,omitempty"`
	ServerReconnectMaxTime     caddy.Duration `json:"server_reconnect_max_time,omitempty"`

	TrackedParameters []strutil.CIString `json:"tracked_parameters,omitempty"`

	RawCritics []json.RawMessage `json:"critics,omitempty" caddy:"namespace=pggat.handlers.pool.critics inline_key=critic"`
	Critics    []pool.Critic     `json:"-"`

	Logger *zap.Logger `json:"-"`
}

func (T Config) Spool() spool.Config {
	return spool.Config{
		PoolerFactory:        new(rob.Factory),
		UsePS:                true,
		UseEQP:               true,
		AcquireTimeout:       time.Duration(T.ClientAcquireTimeout),
		IdleTimeout:          time.Duration(T.ServerIdleTimeout),
		ReconnectInitialTime: time.Duration(T.ServerReconnectInitialTime),
		ReconnectMaxTime:     time.Duration(T.ServerReconnectMaxTime),

		Critics: T.Critics,

		Logger: T.Logger,
	}
}
