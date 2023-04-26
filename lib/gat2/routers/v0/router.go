package routers

import (
	"sync"

	"gfx.cafe/gfx/pggat/lib/gat2"
	"gfx.cafe/gfx/pggat/lib/util/iter"
	"gfx.cafe/gfx/pggat/lib/util/race"
)

type Router struct {
	sinks   []gat2.Sink
	sources []gat2.Source
	mu      sync.RWMutex
}

func NewRouter(sinks []gat2.Sink, sources []gat2.Source) *Router {
	return &Router{
		sinks:   sinks,
		sources: sources,
	}
}

func (T *Router) removesrc(idx int) gat2.Source {
	T.mu.Lock()
	defer T.mu.Unlock()
	source := T.sources[idx]
	for i := idx; i < len(T.sources)-1; i++ {
		T.sources[i] = T.sources[i+1]
	}
	T.sources = T.sources[:len(T.sources)-1]
	return source
}

// srcdead should be called to clean up resources related to a source when the source dies
func (T *Router) srcdead(idx int) {
	source := T.removesrc(idx)

	for _, sink := range T.sinks {
		sink.KillSource(source)
	}
}

// srcrecv is basically a huge select statement on all clients.Out()
func (T *Router) srcrecv() (work gat2.Work, idx int, ok bool) {
	T.mu.RLock()
	defer T.mu.RUnlock()
	if len(T.sources) == 0 {
		return nil, -1, true
	}

	// receive work
	return race.Recv(
		iter.Map(
			iter.Slice(T.sources),
			func(source gat2.Source) <-chan gat2.Work {
				return source.Out()
			},
		),
	)
}

// recv receives the next unit of work from the sources
func (T *Router) recv() gat2.Work {
	for {
		work, idx, ok := T.srcrecv()
		if ok {
			return work
		}

		// T.sources[idx] died, remove it
		T.srcdead(idx)
	}
}

// send tries to get a unit of work done
func (T *Router) send(work gat2.Work) {
	// send work
	race.Send(
		iter.Flatten(
			iter.Map(
				iter.Slice(T.sinks),
				func(sink gat2.Sink) iter.Iter[chan<- gat2.Work] {
					return sink.Route(work)
				},
			),
		),
		work,
	)
}

func (T *Router) route() {
	work := T.recv()
	if work == nil {
		return
	}
	T.send(work)
}

func (T *Router) Run() {
	for {
		T.route()
	}
}
