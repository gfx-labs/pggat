package gat2

import (
	"github.com/google/uuid"

	"gfx.cafe/gfx/pggat/lib/util/iter"
)

type Sink interface {
	ID() uuid.UUID

	// Route will return an iter.Iter of channels equipped to handle the Work.
	// If Sink dies, it should return iter.Empty. If one of the underlying channels dies, they should remain open
	// but not accept work.
	Route(Work) iter.Iter[chan<- Work]

	// KillSource will be called when a source dies. This can be used to free resources related to the Source
	KillSource(Source)
}
