package basic

import (
	"encoding/json"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/poolers/lifo"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/poolers/rob"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/spool"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type ParameterStatusSync string

const (
	// ParameterStatusSyncNone does not attempt to sync parameter status.
	ParameterStatusSyncNone ParameterStatusSync = ""
	// ParameterStatusSyncInitial assumes both client and server have their initial status before syncing.
	// Use in session pooling for lower latency
	ParameterStatusSyncInitial = "initial"
	// ParameterStatusSyncDynamic will track parameter status and ensure they are synced correctly.
	// Use in transaction pooling
	ParameterStatusSyncDynamic = "dynamic"
)

type Config struct {
	RawPoolerFactory json.RawMessage `json:"pooler" caddy:"namespace=pggat.handlers.pool.poolers inline_key=pooler"`

	PoolerFactory pool.PoolerFactory `json:"-"`

	// ReleaseAfterTransaction toggles whether servers should be released and re acquired after each transaction.
	// Use false for lower latency
	// Use true for better balancing
	ReleaseAfterTransaction bool `json:"release_after_transaction,omitempty"`

	// ParameterStatusSync is the parameter syncing mode
	ParameterStatusSync ParameterStatusSync `json:"parameter_status_sync,omitempty"`

	// ExtendedQuerySync controls whether prepared statements and portals should be tracked and synced before use.
	// Use false for lower latency
	// Use true for transaction pooling
	ExtendedQuerySync bool `json:"extended_query_sync,omitempty"`

	ServerResetQuery string `json:"server_reset_query,omitempty"`

	// ServerIdleTimeout defines how long a server may be idle before it is disconnected
	ServerIdleTimeout caddy.Duration `json:"server_idle_timeout,omitempty"`

	// ServerReconnectInitialTime defines how long to wait initially before attempting a server reconnect
	// 0 = disable, don't retry
	ServerReconnectInitialTime caddy.Duration `json:"server_reconnect_initial_time,omitempty"`

	// ServerReconnectMaxTime defines the max amount of time to wait before attempting a server reconnect
	// 0 = disable, back off infinitely
	ServerReconnectMaxTime caddy.Duration `json:"server_reconnect_max_time,omitempty"`

	// TrackedParameters are parameters which should be synced by updating the server, not the client.
	TrackedParameters []strutil.CIString `json:"tracked_parameters,omitempty"`

	Logger *zap.Logger `json:"-"`
}

func (T Config) Spool() spool.Config {
	return spool.Config{
		PoolerFactory:        T.PoolerFactory,
		UsePS:                T.ParameterStatusSync == ParameterStatusSyncDynamic,
		UseEQP:               T.ExtendedQuerySync,
		ResetQuery:           T.ServerResetQuery,
		IdleTimeout:          time.Duration(T.ServerIdleTimeout),
		ReconnectInitialTime: time.Duration(T.ServerReconnectInitialTime),
		ReconnectMaxTime:     time.Duration(T.ServerReconnectMaxTime),

		Logger: T.Logger,
	}
}

var Session = Config{
	RawPoolerFactory: caddyconfig.JSONModuleObject(
		new(lifo.Factory),
		"pooler",
		"lifo",
		nil,
	),
	PoolerFactory:           new(lifo.Factory),
	ReleaseAfterTransaction: false,
	ParameterStatusSync:     ParameterStatusSyncInitial,
	ExtendedQuerySync:       false,
	ServerResetQuery:        "DISCARD ALL",
}

var Transaction = Config{
	RawPoolerFactory: caddyconfig.JSONModuleObject(
		new(rob.Factory),
		"pooler",
		"rob",
		nil,
	),
	PoolerFactory:           new(rob.Factory),
	ReleaseAfterTransaction: true,
	ParameterStatusSync:     ParameterStatusSyncDynamic,
	ExtendedQuerySync:       true,
}
