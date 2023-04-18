package session

import (
	"gfx.cafe/gfx/pggat/lib/gat2/request"
	"gfx.cafe/gfx/pggat/lib/gat2/sink"
	"gfx.cafe/gfx/pggat/lib/gat2/source"
)

type Pool struct {
	in    chan request.Request
	free  []sink.Sink
	inuse map[source.Source]sink.Sink
}

func NewPool(sinks []sink.Sink) *Pool {
	pool := &Pool{
		in:    make(chan request.Request),
		free:  sinks,
		inuse: make(map[source.Source]sink.Sink),
	}
	go pool.run()
	return pool
}

func (T *Pool) gc() {
	for src, s := range T.inuse {
		select {
		case <-src.Closed():
			delete(T.inuse, src)
			T.free = append(T.free, s)
		default:
		}
	}
}

func (T *Pool) usePool(src source.Source) sink.Sink {
	s, ok := T.inuse[src]
	if ok {
		return s
	}

	// collect no longer in use pools
	T.gc()

	if len(T.free) == 0 {
		// TODO(garet) this should just error
		panic("no free pools")
	}

	// steal from free
	s = T.free[len(T.free)-1]
	T.free = T.free[:len(T.free)-1]

	// assign to inuse
	T.inuse[src] = s
	return s
}

func (T *Pool) handle(req request.Request) {
	src := req.Source()
	T.usePool(src).In() <- req
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
