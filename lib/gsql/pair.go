package gsql

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/util/mio"
)

func NewPair() (*fed.Conn, *fed.Conn) {
	conn := new(mio.Conn)
	inward := fed.NewConn(mio.InwardConn{Conn: conn})
	inward.Ready = true
	outward := fed.NewConn(mio.OutwardConn{Conn: conn})
	outward.Ready = true

	return inward, outward
}
