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

	UseOtelTracing bool
	UsePacketTracing bool

	ResetQuery string

	AcquireTimeout time.Duration

	IdleTimeout time.Duration

	ReconnectInitialTime time.Duration
	ReconnectMaxTime     time.Duration

	Critics []pool.Critic

	Logger *zap.Logger
}
