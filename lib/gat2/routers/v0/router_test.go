package routers

import (
	"log"
	"testing"
	"time"

	"github.com/google/uuid"

	"gfx.cafe/gfx/pggat/lib/gat2"
	"gfx.cafe/gfx/pggat/lib/gat2/pools"
	"gfx.cafe/gfx/pggat/lib/util/iter"
)

type DummyWork struct {
	id  uuid.UUID
	src gat2.Source
}

func NewDummyWork(src gat2.Source) *DummyWork {
	return &DummyWork{
		id:  uuid.New(),
		src: src,
	}
}

func (T *DummyWork) ID() uuid.UUID {
	return T.id
}

func (T *DummyWork) Source() gat2.Source {
	return T.src
}

var _ gat2.Work = (*DummyWork)(nil)

type DummySink struct {
	id uuid.UUID
	in chan gat2.Work
}

func NewDummySink() *DummySink {
	s := &DummySink{
		id: uuid.New(),
		in: make(chan gat2.Work),
	}
	go func() {
		for {
			w := <-s.in
			log.Println("received work", w.ID())
		}
	}()
	return s
}

func (T *DummySink) ID() uuid.UUID {
	return T.id
}

func (T *DummySink) Route(_ gat2.Work) iter.Iter[chan<- gat2.Work] {
	return iter.Single[chan<- gat2.Work](T.in)
}

func (T *DummySink) KillSource(_ gat2.Source) {}

var _ gat2.Sink = (*DummySink)(nil)

type DummySource struct {
	id  uuid.UUID
	out chan gat2.Work
}

func NewDummySource() *DummySource {
	src := &DummySource{
		id:  uuid.New(),
		out: make(chan gat2.Work),
	}
	return src
}

func (T *DummySource) QueueWork() {
	go func() {
		T.out <- NewDummyWork(T)
	}()
}

func (T *DummySource) ID() uuid.UUID {
	return T.id
}

func (T *DummySource) Out() <-chan gat2.Work {
	return T.out
}

func (T *DummySource) Close() {
	close(T.out)
}

var _ gat2.Source = (*DummySource)(nil)

func TestRouter(t *testing.T) {
	s1 := NewDummySource()
	s2 := NewDummySource()
	router := NewRouter(
		[]gat2.Sink{
			pools.NewSession([]gat2.Sink{
				NewDummySink(),
			}),
		},
		[]gat2.Source{
			s1,
			s2,
		},
	)

	s1.QueueWork()
	router.route()
	s1.Close()
	s2.QueueWork()
	router.route()

	<-time.After(1 * time.Second)
}
