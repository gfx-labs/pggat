package sink

import (
	"gfx.cafe/gfx/pggat/lib/gat2/request"
)

// Sink is usually a server or pool. This object should have some way to fulfil requests.
type Sink interface {
	// In receives pending requests and fulfills them
	In() chan<- request.Request
}
