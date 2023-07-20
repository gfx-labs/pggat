package transaction

import (
	"log"
	"net"

	"github.com/google/uuid"

	"pggat2/lib/bouncer/backends/v0"
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

func (T *Pool) AddRecipe(name string, recipe gat.Recipe) {
	for i := 0; i < recipe.MinConnections; i++ {
		conn, err := net.Dial("tcp", recipe.Address)
		if err != nil {
			_ = conn.Close()
			// TODO(garet) do something here
			log.Printf("Failed to connect to %s: %v", recipe.Address, err)
			continue
		}
		rw := zap.WrapIOReadWriter(conn)
		eqps := eqp.NewServer()
		pss := ps.NewServer()
		mw := interceptor.NewInterceptor(
			rw,
			eqps,
			pss,
		)
		err2 := backends.Accept(mw, recipe.User, recipe.Password, recipe.Database)
		if err2 != nil {
			_ = conn.Close()
			// TODO(garet) do something here
			log.Printf("Failed to connect to %s: %v", recipe.Address, err2)
			continue
		}
		sink := &Conn{
			pool: T,
			rw:   mw,
			eqp:  eqps,
			ps:   pss,
		}
		id := T.s.AddSink(0, sink)
		sink.id = id
	}
}

func (T *Pool) remove(id uuid.UUID) {
	T.s.RemoveSink(id)
}

func (T *Pool) RemoveRecipe(name string) {
	// TODO(garet) implement
	panic("not implemented")
}

func (T *Pool) Serve(client zap.ReadWriter) {
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
	for {
		if err := buffer.Buffer(); err != nil {
			_ = client.Close()
			break
		}
		source.Do(0, Work{
			rw:  buffer,
			eqp: eqpc,
			ps:  psc,
		})
	}
}

func (T *Pool) ReadSchedulerMetrics(metrics *rob.Metrics) {
	T.s.ReadMetrics(metrics)
}

var _ gat.Pool = (*Pool)(nil)
