package basic

import (
	"encoding/json"

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
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
