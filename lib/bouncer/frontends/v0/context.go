package frontends

import "gfx.cafe/gfx/pggat/lib/fed"

type acceptContext struct {
	Packet  fed.Packet
	Conn    fed.ReadWriter
	Options acceptOptions
}

type authenticateContext struct {
	Packet  fed.Packet
	Conn    fed.ReadWriter
	Options authenticateOptions
}
