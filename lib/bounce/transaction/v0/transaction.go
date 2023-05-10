package transaction

import (
	"pggat2/lib/bounce"
	"pggat2/lib/pnet"
)

var Bouncer = bouncer{}

type bouncer struct{}

func (bouncer) Bounce(client, server pnet.ReadWriter) {
	// TODO implement me
	panic("implement me")
}

var _ bounce.Bouncer = bouncer{}
