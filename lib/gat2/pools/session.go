package pools

import (
	"sync"

	"github.com/google/uuid"

	"gfx.cafe/gfx/pggat/lib/gat2"
	"gfx.cafe/gfx/pggat/lib/util/iter"
)

type Session struct {
	id uuid.UUID

	free  []gat2.Sink
	inuse map[uuid.UUID]gat2.Sink
	mu    sync.RWMutex
}

func NewSession(sinks []gat2.Sink) *Session {
	return &Session{
		id: uuid.New(),

		free:  sinks,
		inuse: make(map[uuid.UUID]gat2.Sink),
	}
}

func (T *Session) ID() uuid.UUID {
	return T.id
}

func (T *Session) bySource(src gat2.Source) gat2.Sink {
	T.mu.RLock()
	defer T.mu.RUnlock()
	sink, _ := T.inuse[src.ID()]
	return sink
}

func (T *Session) cleanup(src gat2.Source) {
	id := src.ID()

	T.mu.Lock()
	defer T.mu.Unlock()

	sink := T.inuse[id]
	delete(T.inuse, id)
	T.free = append(T.free, sink)
}

func (T *Session) assign(src gat2.Source) gat2.Sink {
	T.mu.Lock()
	defer T.mu.Unlock()

	if len(T.free) > 0 {
		// just grab free
		sink := T.free[len(T.free)-1]
		T.free = T.free[:len(T.free)-1]
		T.inuse[src.ID()] = sink

		return sink
	}

	return nil
}

func (T *Session) Route(w gat2.Work) iter.Iter[chan<- gat2.Work] {
	src := w.Source()
	sink := T.bySource(src)
	if sink != nil {
		return sink.Route(w)
	}
	sink = T.assign(src)
	if sink != nil {
		return sink.Route(w)
	}
	return iter.Empty[chan<- gat2.Work]()
}

func (T *Session) KillSource(source gat2.Source) {
	id := source.ID()

	T.mu.Lock()
	defer T.mu.Unlock()

	sink, ok := T.inuse[id]
	if !ok {
		return
	}
	delete(T.inuse, id)
	sink.KillSource(source)

	// return sink to free pool
	T.free = append(T.free, sink)
}

var _ gat2.Sink = (*Session)(nil)
