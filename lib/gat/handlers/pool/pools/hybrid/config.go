package hybrid

import (
	"time"

	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/poolers/rob"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/spool"
)

type Config struct {
	Logger *zap.Logger `json:"-"`
}

func (T Config) Spool() spool.Config {
	return spool.Config{
		PoolerFactory:        new(rob.Factory),
		UsePS:                true,
		UseEQP:               true,
		IdleTimeout:          5 * time.Minute,
		ReconnectInitialTime: 5 * time.Second,
		ReconnectMaxTime:     1 * time.Minute,

		Logger: T.Logger,
	}
}
