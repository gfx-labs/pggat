package frontends

import "gfx.cafe/gfx/pggat/lib/fed"

type acceptContext struct {
	Conn    *fed.Conn
	Options acceptOptions
}

type authenticateContext struct {
	Conn    *fed.Conn
	Options authenticateOptions
}
