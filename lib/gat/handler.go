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
	Handle(ctx context.Context, conn *fed.Conn) error
}

type CancellableHandler interface {
	Handler

	Cancel(ctx context.Context, key fed.BackendKey)
}

type MetricsHandler interface {
	Handler

	ReadMetrics(metrics *metrics.Handler)
}
