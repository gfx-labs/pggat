package gat

import (
	"context"

	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
)

type Client interface {
	Send(pkt protocol.Packet) error
	Recv() <-chan protocol.Packet
}

type ConnectionPool interface {
	GetUser() *config.User
	GetServerInfo() []*protocol.ParameterStatus
	Query(client Client, ctx context.Context, query string) (context.Context, error)
	CallFunction(client Client, ctx context.Context, payload *protocol.FunctionCall) (context.Context, error)
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
