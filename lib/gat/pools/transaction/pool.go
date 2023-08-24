package transaction

import (
	"time"

	"github.com/google/uuid"

	"pggat2/lib/bouncer"
	"pggat2/lib/gat"
	"pggat2/lib/middleware/interceptor"
	"pggat2/lib/middleware/middlewares/eqp"
	"pggat2/lib/middleware/middlewares/ps"
	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v1"
)

type Pool struct {
	config Config
	s      schedulers.Scheduler
}

func NewPool(config Config) *Pool {
	pool := &Pool{
		config: config,
		s:      schedulers.MakeScheduler(),
	}

	return pool
}

func (T *Pool) AddServer(server bouncer.Conn) uuid.UUID {
	eqps := eqp.NewServer()
	pss := ps.NewServer(server.InitialParameters)
	server.RW = interceptor.NewInterceptor(
		server.RW,
		eqps,
		pss,
	)
	sink := &Conn{
		b:   server,
		eqp: eqps,
		ps:  pss,
	}
	return T.s.AddWorker(0, sink)
}

func (T *Pool) GetServer(id uuid.UUID) bouncer.Conn {
	conn := T.s.GetWorker(id)
	if conn == nil {
		return bouncer.Conn{}
	}
	return conn.(*Conn).b
}

func (T *Pool) RemoveServer(id uuid.UUID) bouncer.Conn {
	conn := T.s.RemoveWorker(id)
	if conn == nil {
		return bouncer.Conn{}
	}
	return conn.(*Conn).b
}

func (T *Pool) Serve(ctx *gat.Context, client bouncer.Conn) {
	source := T.s.NewSource()
	eqpc := eqp.NewClient()
	defer eqpc.Done()
	psc := ps.NewClient(client.InitialParameters)
	c := interceptor.NewInterceptor(
		client.RW,
		eqpc,
		psc,
	)
	robCtx := rob.Context{
		OnWait: ctx.OnWait,
	}

	for {
		packet, err := c.ReadPacket(true)
		if err != nil {
			break
		}

		source.Do(&robCtx, Work{
			rw:                c,
			initialPacket:     packet,
			eqp:               eqpc,
			ps:                psc,
			trackedParameters: T.config.TrackedParameters,
		})
	}
	_ = c.Close()
}

func (T *Pool) LookupCorresponding(key [8]byte) (uuid.UUID, [8]byte, bool) {
	// TODO(garet)
	return uuid.Nil, [8]byte{}, false
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
		_ = conn.b.RW.Close()
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
