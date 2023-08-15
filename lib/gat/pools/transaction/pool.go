package transaction

import (
	"time"

	"github.com/google/uuid"

	"pggat2/lib/gat"
	"pggat2/lib/middleware/interceptor"
	"pggat2/lib/middleware/middlewares/eqp"
	"pggat2/lib/middleware/middlewares/ps"
	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v1"
	"pggat2/lib/util/strutil"
	"pggat2/lib/zap"
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

func (T *Pool) AddServer(server zap.ReadWriter, parameters map[strutil.CIString]string) uuid.UUID {
	eqps := eqp.NewServer()
	pss := ps.NewServer(parameters)
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

func (T *Pool) Serve(ctx *gat.Context, client zap.ReadWriter, _ map[strutil.CIString]string) {
	source := T.s.NewSource()
	eqpc := eqp.NewClient()
	defer eqpc.Done()
	psc := ps.NewClient()
	client = interceptor.NewInterceptor(
		client,
		eqpc,
		psc,
	)
	robCtx := rob.Context{
		OnWait: ctx.OnWait,
	}

	packet := zap.NewPacket()
	defer packet.Done()

	for {
		if err := client.Read(packet); err != nil {
			break
		}

		source.Do(&robCtx, Work{
			rw:            client,
			initialPacket: packet,
			eqp:           eqpc,
			ps:            psc,
		})
	}
	_ = client.Close()
}

func (T *Pool) ScaleDown(amount int) (remaining int) {
	remaining = amount

	for i := 0; i < amount; i++ {
		id, idle := T.s.GetIdleWorker()
		if id == uuid.Nil || idle == (time.Time{}) {
			break
		}
		worker := T.s.RemoveWorker(id)
		if worker == nil {
			i--
			continue
		}
		conn := worker.(*Conn)
		_ = conn.rw.Close()
		remaining--
	}

	return
}

func (T *Pool) IdleSince() time.Time {
	_, idle := T.s.GetIdleWorker()
	return idle
}

func (T *Pool) ReadSchedulerMetrics(metrics *rob.Metrics) {
	T.s.ReadMetrics(metrics)
}

var _ gat.RawPool = (*Pool)(nil)
