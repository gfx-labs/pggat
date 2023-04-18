package request

import (
	"gfx.cafe/gfx/pggat/lib/gat2/request"
	"gfx.cafe/gfx/pggat/lib/gat2/sink"
	"gfx.cafe/gfx/pggat/lib/util/race"
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
	if len(T.sinks) == 0 {
		// TODO(garet) this should just error
		panic("no free pools")
	}
	// choose a random sink to wait for
	ok := race.Send(func(i int) (chan<- request.Request, bool) {
		if i >= len(T.sinks) {
			return nil, false
		}
		return T.sinks[i].In(), true
	}, req)
	if !ok {
		panic("failed to send req to pool")
	}
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
