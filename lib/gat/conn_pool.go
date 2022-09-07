package gat

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
)

type ConnectionPool interface {
	GetUser() *config.User
	GetServerInfo() []*protocol.ParameterStatus
	Query(client Client, ctx context.Context, query string) (context.Context, error)
	CallFunction(client Client, ctx context.Context, payload *protocol.FunctionCall) (context.Context, error)
}
