package session

import (
	"pggat2/lib/bounce"
	"pggat2/lib/pnet"
)

var Bouncer = bouncer{}

type bouncer struct{}

func (bouncer) Bounce(client, server pnet.ReadWriter) {
	// bounce from client to server until client disconnects
}

var _ bounce.Bouncer = bouncer{}
