package transaction

import (
	"pggat2/lib/middleware/middlewares/eqp"
	"pggat2/lib/middleware/middlewares/ps"
	"pggat2/lib/util/strutil"
	"pggat2/lib/zap"
)

type Work struct {
	rw                zap.Conn
	initialPacket     zap.Packet
	eqp               *eqp.Client
	ps                *ps.Client
	trackedParameters []strutil.CIString
}
