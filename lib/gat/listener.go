package gat

import (
	"gfx.cafe/gfx/pggat/lib/bouncer/frontends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
)

type AcceptedConn struct {
	Conn   fed.Conn
	Params frontends.AcceptParams
}

type Listener interface {
	Module

	Accept() []<-chan AcceptedConn
}
