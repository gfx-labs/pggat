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
	Query(ctx context.Context, client Client, query string) error
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
