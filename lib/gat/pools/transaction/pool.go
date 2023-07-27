package transaction

import (
	"github.com/google/uuid"

	"pggat2/lib/gat"
	"pggat2/lib/middleware/interceptor"
	"pggat2/lib/middleware/middlewares/eqp"
	"pggat2/lib/middleware/middlewares/ps"
	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v1"
	"pggat2/lib/zap"
	"pggat2/lib/zap/zapbuf"
)

type Pool struct {
	s schedulers.Scheduler
}

func NewPool() *Pool {
	pool := &Pool{
		s: schedulers.MakeScheduler(),
	}

	return pool
}

func (T *Pool) AddServer(server zap.ReadWriter) uuid.UUID {
	eqps := eqp.NewServer()
	pss := ps.NewServer()
	mw := interceptor.NewInterceptor(
		server,
		eqps,
		pss,
	)
	sink := &Conn{
		rw:  mw,
		eqp: eqps,
		ps:  pss,
	}
	return T.s.AddWorker(0, sink)
}

func (T *Pool) GetServer(id uuid.UUID) zap.ReadWriter {
	conn := T.s.GetWorker(id)
	if conn == nil {
		return nil
	}
	return conn.(*Conn).rw
}

func (T *Pool) RemoveServer(id uuid.UUID) zap.ReadWriter {
	conn := T.s.RemoveWorker(id)
	if conn == nil {
		return nil
	}
	return conn.(*Conn).rw
}

func (T *Pool) Serve(ctx *gat.Context, client zap.ReadWriter) {
	source := T.s.NewSource()
	eqpc := eqp.NewClient()
	defer eqpc.Done()
	psc := ps.NewClient()
	client = interceptor.NewInterceptor(
		client,
		eqpc,
		psc,
	)
	buffer := zapbuf.NewBuffer(client)
	defer buffer.Done()
	robCtx := rob.Context{
		OnWait: ctx.OnWait,
	}
	for {
		if err := buffer.Buffer(); err != nil {
			break
		}
		source.Do(&robCtx, Work{
			rw:  buffer,
			eqp: eqpc,
			ps:  psc,
		})
	}
	_ = client.Close()
}

func (T *Pool) ReadSchedulerMetrics(metrics *rob.Metrics) {
	T.s.ReadMetrics(metrics)
}

var _ gat.RawPool = (*Pool)(nil)
