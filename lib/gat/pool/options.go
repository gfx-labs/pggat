package pool

import (
	"time"

	"gfx.cafe/gfx/pggat/lib/auth"
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

type Options struct {
	Credentials auth.Credentials

	NewPooler func() Pooler
	// ReleaseAfterTransaction toggles whether servers should be released and re acquired after each transaction.
	// Use false for lower latency
	// Use true for better balancing
	ReleaseAfterTransaction bool

	ServerResetQuery string
	// ServerIdleTimeout defines how long a server may be idle before it is disconnected
	ServerIdleTimeout time.Duration

	// ServerReconnectInitialTime defines how long to wait initially before attempting a server reconnect
	// 0 = disable, don't retry
	ServerReconnectInitialTime time.Duration
	// ServerReconnectMaxTime defines the max amount of time to wait before attempting a server reconnect
	// 0 = disable, back off infinitely
	ServerReconnectMaxTime time.Duration

	// ParameterStatusSync is the parameter syncing mode
	ParameterStatusSync ParameterStatusSync
	// TrackedParameters are parameters which should be synced by updating the server, not the client.
	TrackedParameters []strutil.CIString

	// ExtendedQuerySync controls whether prepared statements and portals should be tracked and synced before use.
	// Use false for lower latency
	// Use true for transaction pooling
	ExtendedQuerySync bool
}
