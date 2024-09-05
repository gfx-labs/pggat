package basic

import (
	"encoding/json"
	"errors"
	"strings"
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

type TracingOption int

const (
	// TracingOptionDisabled indicates tracing is disabled
	TracingOptionDisabled TracingOption = iota
	// TracingOptionClient indicates tracing is enabled for client connections
	TracingOptionClient = 1 << (iota - 1)
	// TracingOptionServer indicates tracing is enabled for server connections
	TracingOptionServer = 1 << (iota - 1)
	// TracingOptionClientAndServer indicates tracing is enabled for both
	// client and server connections
	TracingOptionClientAndServer = TracingOptionClient | TracingOptionServer
)

func MapTracingOption(s string) (opt TracingOption, err error) {
	switch strings.ToLower(s) {
	case "disable", "disabled", "none", "off":
	case "client":
		opt = TracingOptionClient
	case "server":
		opt = TracingOptionServer
	case "client-and-server", "both", "all":
		opt = TracingOptionClientAndServer
	default:
		err = errors.New("unknown tracing option: " + s)
	}

	return
}

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

	// PacketTracingOption enables/disables packet debug tracing for client and/or
	// server connections
	PacketTracingOption TracingOption `json:"packet_tracing_option,omitempty"`

	// OtelTracingOption enables/disables Open Telemetry tracing for client and/or
	// server connections
	OtelTracingOption TracingOption `json:"otel_tracing_option,omitempty"`

	ServerResetQuery string `json:"server_reset_query,omitempty"`

	// ClientAcquireTimeout defines how long a client may be in AWAITING_SERVER state before it is disconnected
	ClientAcquireTimeout caddy.Duration `json:"client_acquire_timeout,omitempty"`

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

	RawCritics []json.RawMessage `json:"critics,omitempty" caddy:"namespace=pggat.handlers.pool.critics inline_key=critic"`
	Critics    []pool.Critic     `json:"-"`

	Logger *zap.Logger `json:"-"`
}

func (T Config) Spool() spool.Config {
	return spool.Config{
		PoolerFactory:        T.PoolerFactory,
		UsePS:                T.ParameterStatusSync == ParameterStatusSyncDynamic,
		UseEQP:               T.ExtendedQuerySync,
		UseOtelTracing:       (T.OtelTracingOption & TracingOptionServer) != 0,
		UsePacketTracing:     (T.PacketTracingOption & TracingOptionServer) != 0,
		ResetQuery:           T.ServerResetQuery,
		AcquireTimeout:       time.Duration(T.ClientAcquireTimeout),
		IdleTimeout:          time.Duration(T.ServerIdleTimeout),
		ReconnectInitialTime: time.Duration(T.ServerReconnectInitialTime),
		ReconnectMaxTime:     time.Duration(T.ServerReconnectMaxTime),

		Critics: T.Critics,

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
	OtelTracingOption:       TracingOptionClient,
	PacketTracingOption:     TracingOptionDisabled,
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
	OtelTracingOption:       TracingOptionClient,
	PacketTracingOption:     TracingOptionDisabled,
}
