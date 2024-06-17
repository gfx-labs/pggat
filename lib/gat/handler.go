package gat

import (
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
	Route(conn *fed.Conn) error
}
type RouterFunc func(conn *fed.Conn) error

func (R RouterFunc) Route(conn *fed.Conn) error {
	return R(conn)
}

type CancellableHandler interface {
	Handler

	Cancel(key fed.BackendKey)
}

type MetricsHandler interface {
	Handler

	ReadMetrics(metrics *metrics.Handler)
}
