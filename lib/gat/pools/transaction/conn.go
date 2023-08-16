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

func (T *Conn) Do(ctx *rob.Context, work any) {
	job := work.(Work)

	// sync parameters
	ps.Sync(job.trackedParameters, job.rw, job.ps, T.rw, T.ps)

	T.eqp.SetClient(job.eqp)
	clientErr, serverErr := bouncers.Bounce(job.rw, T.rw, job.initialPacket)
	if clientErr != nil || serverErr != nil {
		_ = job.rw.Close()
		if serverErr != nil {
			_ = T.rw.Close()
			ctx.Remove()
		}
	}
	return
}

var _ rob.Worker = (*Conn)(nil)
