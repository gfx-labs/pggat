package source

import (
	"gfx.cafe/gfx/pggat/lib/gat2/request"
)

// Source is usually a client. This object should generate requests to be fulfilled by sinks.
type Source interface {
	// Out sends pending requests to be fulfilled by a sink
	Out() <-chan request.Request

	Closed() <-chan struct{}
}
