package transaction

import (
	"pggat2/lib/bouncer/bouncers/v2"
	"pggat2/lib/middleware/middlewares/eqp"
	"pggat2/lib/middleware/middlewares/ps"
	"pggat2/lib/rob"
	"pggat2/lib/zap"
)

type Conn struct {
	rw  zap.ReadWriter
	eqp *eqp.Server
	ps  *ps.Server
}

func (T Conn) Do(_ rob.Constraints, work any) {
	job := work.(Work)
	job.ps.SetServer(T.ps)
	T.eqp.SetClient(job.eqp)
	bouncers.Bounce(job.rw, T.rw)
	return
}

var _ rob.Worker = Conn{}
