package gat

import (
	"context"
	"net"
	"time"

	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
)

type ClientID struct {
	PID       int32
	SecretKey int32
}

type ClientState string

const (
	ClientActive  ClientState = "active"
	ClientWaiting             = "waiting"
)

type Client interface {
	GetId() ClientID

	GetOptions() []protocol.FieldsStartupMessageParameters

	GetPreparedStatement(name string) *protocol.Parse
	GetPortal(name string) *protocol.Bind
	GetCurrentConn() Connection
	SetCurrentConn(conn Connection)

	GetConnectionPool() Pool

	GetState() ClientState
	GetAddress() net.Addr
	GetLocalAddress() net.Addr
	GetConnectTime() time.Time
	GetRequestTime() time.Time
	GetRemotePid() int

	// sharding
	SetRequestedShard(shard int)
	UnsetRequestedShard()
	GetRequestedShard() (int, bool)
	SetShardingKey(key string)
	GetShardingKey() string

	Send(pkt protocol.Packet) error
	Flush() error
	Recv() <-chan protocol.Packet
}

type Gat interface {
	GetVersion() string
	GetConfig() *config.Global
	GetDatabase(name string) Database
	GetDatabases() map[string]Database
	GetClient(id ClientID) Client
	GetClients() []Client
}

type Database interface {
	GetUser(name string) *config.User
	GetRouter() QueryRouter
	GetName() string

	WithUser(name string) Pool
	GetPools() []Pool

	EnsureConfig(c *config.Pool)
}

type QueryRouter interface {
	InferRole(query string) (config.ServerRole, error)
	// TryHandle the client's query string. If we handled it, return true
	TryHandle(client Client, query string) (bool, error)
}

type Pool interface {
	GetUser() *config.User
	GetServerInfo(client Client) []*protocol.ParameterStatus

	GetDatabase() Database

	EnsureConfig(c *config.Pool)

	OnDisconnect(client Client)

	// extended queries
	Describe(ctx context.Context, client Client, describe *protocol.Describe) error
	Execute(ctx context.Context, client Client, execute *protocol.Execute) error

	// simple queries
	SimpleQuery(ctx context.Context, client Client, query string) error
	Transaction(ctx context.Context, client Client, query string) error
	CallFunction(ctx context.Context, client Client, payload *protocol.FunctionCall) error
}

type ConnectionState string

const (
	ConnectionActive ConnectionState = "active"
	ConnectionIdle                   = "idle"
	ConnectionUsed                   = "used"
	ConnectionTested                 = "tested"
	ConnectionNew                    = "new"
)

type Dialer = func(context.Context, []protocol.FieldsStartupMessageParameters, *config.User, *config.Shard, *config.Server) (Connection, error)

type Connection interface {
	GetServerInfo() []*protocol.ParameterStatus

	GetDatabase() string
	GetState() ConnectionState
	GetHost() string
	GetPort() int
	GetAddress() net.Addr
	GetLocalAddress() net.Addr
	GetConnectTime() time.Time
	GetRequestTime() time.Time
	GetClient() Client
	SetClient(client Client)
	GetRemotePid() int
	GetTLS() string

	// IsCloseNeeded returns whether this connection failed a health check
	IsCloseNeeded() bool
	Close() error

	// actions
	Describe(ctx context.Context, client Client, payload *protocol.Describe) error
	Execute(ctx context.Context, client Client, payload *protocol.Execute) error
	CallFunction(ctx context.Context, client Client, payload *protocol.FunctionCall) error
	SimpleQuery(ctx context.Context, client Client, payload string) error
	Transaction(ctx context.Context, client Client, payload string) error

	// Cancel the current running query
	Cancel() error
}
