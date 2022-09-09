package gat

import (
	"context"

	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
)

type Client interface {
	GetPreparedStatement(name string) *protocol.Parse
	GetPortal(name string) *protocol.Bind

	Send(pkt protocol.Packet) error
	Recv() <-chan protocol.Packet
}

type ConnectionPool interface {
	GetUser() *config.User
	GetServerInfo() []*protocol.ParameterStatus

	// extended queries
	Describe(ctx context.Context, client Client, describe *protocol.Describe) error
	Execute(ctx context.Context, client Client, execute *protocol.Execute) error

	// simple queries
	SimpleQuery(ctx context.Context, client Client, query string) error
	Transaction(ctx context.Context, client Client, query string) error
	CallFunction(ctx context.Context, client Client, payload *protocol.FunctionCall) error
}

type Gat interface {
	GetPool(name string) (Pool, error)
}

type Pool interface {
	GetUser(name string) (*config.User, error)
	GetRouter() QueryRouter
	WithUser(name string) (ConnectionPool, error)
}

type QueryRouter interface {
	InferRole(query string) (config.ServerRole, error)
}
