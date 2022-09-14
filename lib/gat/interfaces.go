package gat

import (
	"context"
	"errors"
	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"time"
)

type ClientID struct {
	PID       int32
	SecretKey int32
}

type Client interface {
	GetPreparedStatement(name string) *protocol.Parse
	GetPortal(name string) *protocol.Bind
	GetCurrentConn() (Connection, error)
	SetCurrentConn(conn Connection)

	State() string
	Addr() string
	Port() int
	LocalAddr() string
	LocalPort() int
	ConnectTime() time.Time
	RequestTime() time.Time
	Wait() time.Duration
	RemotePid() int

	Send(pkt protocol.Packet) error
	Flush() error
	Recv() <-chan protocol.Packet
}

type Gat interface {
	Version() string
	Config() *config.Global
	GetPool(name string) (Pool, error)
	Pools() map[string]Pool
	GetClient(id ClientID) (Client, error)
	Clients() []Client
}

var UserNotFound = errors.New("user not found")

type Pool interface {
	GetUser(name string) (*config.User, error)
	GetRouter() QueryRouter

	WithUser(name string) (ConnectionPool, error)
	ConnectionPools() []ConnectionPool

	Stats() *PoolStats

	EnsureConfig(c *config.Pool)
}

type QueryRouter interface {
	InferRole(query string) (config.ServerRole, error)
}

type ConnectionPool interface {
	GetUser() *config.User
	GetServerInfo() []*protocol.ParameterStatus

	Shards() []Shard

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
	Primary() Connection
	Replicas() []Connection
	Choose(role config.ServerRole) Connection
}

type Connection interface {
	GetServerInfo() []*protocol.ParameterStatus

	GetDatabase() string
	State() string
	Address() string
	Port() int
	LocalAddr() string
	LocalPort() int
	ConnectTime() time.Time
	RequestTime() time.Time
	Wait() time.Duration
	CloseNeeded() bool
	Client() Client
	SetClient(client Client)
	RemotePid() int
	TLS() string

	// actions
	Describe(client Client, payload *protocol.Describe) error
	Execute(client Client, payload *protocol.Execute) error
	CallFunction(client Client, payload *protocol.FunctionCall) error
	SimpleQuery(ctx context.Context, client Client, payload string) error
	Transaction(ctx context.Context, client Client, payload string) error

	// Cancel the current running query
	Cancel() error
}
