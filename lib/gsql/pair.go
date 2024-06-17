package gsql

import (
	"net"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/fed/codecs/netconncodec"
	"gfx.cafe/gfx/pggat/lib/util/mio"
)

func NewPair() (*fed.Conn, *fed.Conn, net.Conn, net.Conn) {
	conn := new(mio.Conn)
	in := mio.InwardConn{Conn: conn}
	out := mio.InwardConn{Conn: conn}
	inward := fed.NewConn(
		netconncodec.NewCodec(
			in,
		),
	)
	inward.Ready = true
	outward := fed.NewConn(
		netconncodec.NewCodec(
			out,
		),
	)
	outward.Ready = true

	return inward, outward, in, out
}
