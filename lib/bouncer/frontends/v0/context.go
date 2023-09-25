package frontends

import "gfx.cafe/gfx/pggat/lib/fed"

type AcceptContext struct {
	Packet  fed.Packet
	Conn    fed.Conn
	Options AcceptOptions
}

type AuthenticateContext struct {
	Packet  fed.Packet
	Conn    fed.Conn
	Options AuthenticateOptions
}
