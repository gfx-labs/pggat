package gat

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"net"
	"time"
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
	GetPreparedStatement(name string) *protocol.Parse
	GetPortal(name string) *protocol.Bind
	GetCurrentConn() Connection
	SetCurrentConn(conn Connection)

	GetConnectionPool() ConnectionPool

	GetState() ClientState
	GetAddress() net.Addr
	GetLocalAddress() net.Addr
	GetConnectTime() time.Time
	GetRequestTime() time.Time
	GetRemotePid() int

	Send(pkt protocol.Packet) error
	Flush() error
	Recv() <-chan protocol.Packet
}

type Gat interface {
	GetVersion() string
	GetConfig() *config.Global
	GetPool(name string) Pool
	GetPools() map[string]Pool
	GetClient(id ClientID) Client
	GetClients() []Client
}

type Pool interface {
	GetUser(name string) *config.User
	GetRouter() QueryRouter

	WithUser(name string) ConnectionPool
	ConnectionPools() []ConnectionPool

	GetStats() *PoolStats

	EnsureConfig(c *config.Pool)
}

type QueryRouter interface {
	InferRole(query string) (config.ServerRole, error)
}

type ConnectionPool interface {
	GetUser() *config.User
	GetServerInfo() []*protocol.ParameterStatus

	GetPool() Pool

	EnsureConfig(c *config.Pool)

	// extended queries
	Describe(ctx context.Context, client Client, describe *protocol.Describe) error
	Execute(ctx context.Context, client Client, execute *protocol.Execute) error

	// simple queries
	SimpleQuery(ctx context.Context, client Client, query string) error
	Transaction(ctx context.Context, client Client, query string) error
	CallFunction(ctx context.Context, client Client, payload *protocol.FunctionCall) error
}

type Shard interface {
	GetPrimary() Connection
	GetReplicas() []Connection
	Choose(role config.ServerRole) Connection

	EnsureConfig(c *config.Shard)
}

type ConnectionState string

const (
	ConnectionActive ConnectionState = "active"
	ConnectionIdle                   = "idle"
	ConnectionUsed                   = "used"
	ConnectionTested                 = "tested"
	ConnectionNew                    = "new"
)

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

	// actions
	Describe(client Client, payload *protocol.Describe) error
	Execute(client Client, payload *protocol.Execute) error
	CallFunction(client Client, payload *protocol.FunctionCall) error
	SimpleQuery(ctx context.Context, client Client, payload string) error
	Transaction(ctx context.Context, client Client, payload string) error

	// Cancel the current running query
	Cancel() error
}
