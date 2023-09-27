package pool

import (
	"gfx.cafe/gfx/pggat/lib/auth"
	"gfx.cafe/gfx/pggat/lib/util/dur"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type ParameterStatusSync int

const (
	// ParameterStatusSyncNone does not attempt to sync parameter status.
	ParameterStatusSyncNone ParameterStatusSync = iota
	// ParameterStatusSyncInitial assumes both client and server have their initial status before syncing.
	// Use in session pooling for lower latency
	ParameterStatusSyncInitial
	// ParameterStatusSyncDynamic will track parameter status and ensure they are synced correctly.
	// Use in transaction pooling
	ParameterStatusSyncDynamic
)

type PoolingOptions struct {
	NewPooler func() Pooler
	// ReleaseAfterTransaction toggles whether servers should be released and re acquired after each transaction.
	// Use false for lower latency
	// Use true for better balancing
	ReleaseAfterTransaction bool

	// ParameterStatusSync is the parameter syncing mode
	ParameterStatusSync ParameterStatusSync

	// ExtendedQuerySync controls whether prepared statements and portals should be tracked and synced before use.
	// Use false for lower latency
	// Use true for transaction pooling
	ExtendedQuerySync bool
}

type ManagementOptions struct {
	ServerResetQuery string `json:"server_reset_query,omitempty"`
	// ServerIdleTimeout defines how long a server may be idle before it is disconnected
	ServerIdleTimeout dur.Duration `json:"server_idle_timeout,omitempty"`

	// ServerReconnectInitialTime defines how long to wait initially before attempting a server reconnect
	// 0 = disable, don't retry
	ServerReconnectInitialTime dur.Duration `json:"server_reconnect_initial_time,omitempty"`
	// ServerReconnectMaxTime defines the max amount of time to wait before attempting a server reconnect
	// 0 = disable, back off infinitely
	ServerReconnectMaxTime dur.Duration `json:"server_reconnect_max_time,omitempty"`

	// TrackedParameters are parameters which should be synced by updating the server, not the client.
	TrackedParameters []strutil.CIString `json:"tracked_parameters,omitempty"`
}

type Options struct {
	Credentials auth.Credentials

	PoolingOptions

	ManagementOptions
}
