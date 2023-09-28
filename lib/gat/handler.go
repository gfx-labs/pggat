package gat

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
)

// Handler handles the Conn
type Handler interface {
	// Handle will attempt to handle the Conn. Return io.EOF for normal disconnection or nil to continue to the next
	// handle. The error will be relayed to the client so there is no need to send it yourself.
	Handle(conn *fed.Conn) error
}

type CancellableHandler interface {
	Handler

	Cancel(key [8]byte)
}

type MetricsHandler interface {
	Handler

	ReadMetrics(metrics *metrics.Handler)
}
