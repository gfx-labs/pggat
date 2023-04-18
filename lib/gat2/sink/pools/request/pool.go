package request

import (
	"gfx.cafe/gfx/pggat/lib/gat2/request"
	"gfx.cafe/gfx/pggat/lib/gat2/sink"
	"math/rand"
)

type Pool struct {
	in    chan request.Request
	sinks []sink.Sink
}

func NewPool(sinks []sink.Sink) *Pool {
	pool := &Pool{
		in:    make(chan request.Request),
		sinks: sinks,
	}
	go pool.run()
	return pool
}

func (T *Pool) handle(req request.Request) {
	for _, s := range T.sinks {
		select {
		case s.In() <- req:
			return
		default:
		}
	}
	// choose a random sink to wait for
	if len(T.sinks) == 0 {
		// TODO(garet) this should just error
		panic("no free pools")
	}
	T.sinks[rand.Intn(len(T.sinks))].In() <- req
}

func (T *Pool) run() {
	for {
		T.handle(<-T.in)
	}
}

func (T *Pool) In() chan<- request.Request {
	return T.in
}

var _ sink.Sink = (*Pool)(nil)
