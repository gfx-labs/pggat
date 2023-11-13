package spool

import (
	"time"

	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
)

type Config struct {
	PoolerFactory pool.PoolerFactory

	// UsePS controls whether to add the ps middleware to servers
	UsePS bool
	// UseEQP controls whether to add the eqp middleware to servers
	UseEQP bool

	ResetQuery string

	IdleTimeout time.Duration

	ReconnectInitialTime time.Duration
	ReconnectMaxTime     time.Duration

	Scorers []pool.Scorer

	Logger *zap.Logger
}
