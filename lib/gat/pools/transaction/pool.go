package transaction

import (
	"net"

	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/gat"
	"pggat2/lib/middleware/interceptor"
	"pggat2/lib/middleware/middlewares/eqp"
	"pggat2/lib/middleware/middlewares/ps"
	"pggat2/lib/rob/schedulers/v1"
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

func (T *Pool) AddRecipe(name string, recipe gat.Recipe) {
	for i := 0; i < recipe.MinConnections; i++ {
		conn, err := net.Dial("tcp", recipe.Address)
		if err != nil {
			// TODO(garet) do something here
			continue
		}
		rw := zap.CombinedReadWriter{
			Reader: zap.IOReader{Reader: conn},
			Writer: zap.IOWriter{Writer: conn},
		}
		eqps := eqp.NewServer()
		pss := ps.NewServer()
		mw := interceptor.NewInterceptor(
			rw,
			eqps,
			pss,
		)
		backends.Accept(mw, recipe.User, recipe.Password, recipe.Database)
		T.s.AddSink(0, Conn{
			rw:  mw,
			eqp: eqps,
			ps:  pss,
		})
	}
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
	for {
		if err := client.Poll(); err != nil {
			break
		}
		source.Do(0, Work{
			rw:  client,
			eqp: eqpc,
			ps:  psc,
		})
	}
}

var _ gat.Pool = (*Pool)(nil)
