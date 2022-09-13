package gat

import (
	"context"

	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
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
	TotalReceived()
	// Total bytes sent over network
	TotalSent()
	// Total time spent doing transactions
	TotalXactTime()
	// Total time spent doing queries
	TotalQueryTime()
	// Total time spent waiting
	TotalWaitTime()
	// Average amount of transactions per second
	AvgXactCount()
	// Average amount of queries per second
	AvgQueryCount()
	// Average bytes received per second
	AvgRecv()
	// Average bytes sent per second
	AvgSent()
	// Average time transactions take
	AvgXactTime()
	// Average time queries take
	AvgQueryTime()
	// Average time waiting for work
	AvgWaitTime()
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
}

type Connection interface {
	GetDatabase() string

	// Cancel the current running query
	Cancel() error
}
