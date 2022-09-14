package gat

import (
	"context"
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
	GetPool(name string) (Pool, error)
	Pools() []Pool
	GetClient(id ClientID) (Client, error)
	Clients() []Client
}

type Pool interface {
	GetUser(name string) (*config.User, error)
	GetRouter() QueryRouter

	WithUser(name string) (ConnectionPool, error)
	ConnectionPools() []ConnectionPool

	Stats() PoolStats

	EnsureConfig(c *config.Pool)
}

type PoolStats interface {
	// Total transactions
	TotalXactCount() int
	// Total queries
	TotalQueryCount() int
	// Total bytes received over network
	TotalReceived() int
	// Total bytes sent over network
	TotalSent() int
	// Total time spent doing transactions (in microseconds)
	TotalXactTime() int
	// Total time spent doing queries (in microseconds)
	TotalQueryTime() int
	// Total time spent waiting (in microseconds)
	TotalWaitTime() int
	// Average amount of transactions per second
	AvgXactCount() float64
	// Average amount of queries per second
	AvgQueryCount() float64
	// Average bytes received per second
	AvgRecv() float64
	// Average bytes sent per second
	AvgSent() float64
	// Average time transactions take (in microseconds)
	AvgXactTime() float64
	// Average time queries take (in microseconds)
	AvgQueryTime() float64
	// Average time waiting for work (in microseconds)
	AvgWaitTime() float64
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
