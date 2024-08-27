package gat

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
)

// Handler handles the Conn
type Handler interface {
	// Handle will attempt to handle the Conn. Return io.EOF for normal disconnection or nil to continue to the next
	// handle. The error will be relayed to the client so there is no need to send it yourself.
	Handle(Router) Router
}

type HandlerFunc func(Router) Router

func (H HandlerFunc) Handle(next Router) Router {
	return H(next)
}

type Router interface {
	Route(ctx context.Context, conn *fed.Conn) error
}
type RouterFunc func(ctx context.Context, conn *fed.Conn) error

func (R RouterFunc) Route(ctx context.Context, conn *fed.Conn) error {
	return R(ctx, conn)
}

type CancellableHandler interface {
	Handler

	Cancel(ctx context.Context, key fed.BackendKey)
}

type MetricsHandler interface {
	Handler

	ReadMetrics(ctx context.Context, metrics *metrics.Handler)
}
