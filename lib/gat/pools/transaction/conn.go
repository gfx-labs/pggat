package transaction

import (
	"pggat2/lib/bouncer"
	"pggat2/lib/bouncer/bouncers/v2"
	"pggat2/lib/middleware/middlewares/eqp"
	"pggat2/lib/middleware/middlewares/ps"
	"pggat2/lib/rob"
)

type Conn struct {
	b   bouncer.Conn
	eqp *eqp.Server
	ps  *ps.Server
}

func (T *Conn) Do(ctx *rob.Context, work any) {
	job := work.(Work)

	var clientErr, serverErr error

	defer func() {
		if clientErr != nil || serverErr != nil {
			_ = job.rw.Close()
			if serverErr != nil {
				_ = T.b.RW.Close()
				ctx.Remove()
			}
		}
	}()

	// sync parameters
	clientErr, serverErr = ps.Sync(job.trackedParameters, job.rw, job.ps, T.b.RW, T.ps)
	if clientErr != nil || serverErr != nil {
		return
	}

	T.eqp.SetClient(job.eqp)
	clientErr, serverErr = bouncers.Bounce(job.rw, T.b.RW, job.initialPacket)
	if clientErr != nil || serverErr != nil {
		return
	}
	return
}

var _ rob.Worker = (*Conn)(nil)
