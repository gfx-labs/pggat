package transaction

import (
	"github.com/google/uuid"

	"pggat2/lib/bouncer/bouncers/v2"
	"pggat2/lib/middleware/middlewares/eqp"
	"pggat2/lib/middleware/middlewares/ps"
	"pggat2/lib/rob"
	"pggat2/lib/zap"
)

type Conn struct {
	pool *Pool
	id   uuid.UUID
	rw   zap.ReadWriter
	eqp  *eqp.Server
	ps   *ps.Server
}

func (T *Conn) Do(_ rob.Constraints, work any) {
	job := work.(Work)
	job.ps.SetServer(T.ps)
	T.eqp.SetClient(job.eqp)
	clientErr, serverErr := bouncers.Bounce(job.rw, T.rw)
	if clientErr != nil || serverErr != nil {
		_ = job.rw.Close()
		if serverErr != nil {
			_ = T.rw.Close()
			T.pool.remove(T.id)
			panic(serverErr)
		}
	}
	return
}

var _ rob.Worker = (*Conn)(nil)
