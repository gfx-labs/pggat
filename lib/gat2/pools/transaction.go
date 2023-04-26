package pools

import (
	"github.com/google/uuid"

	"gfx.cafe/gfx/pggat/lib/gat2"
	"gfx.cafe/gfx/pggat/lib/util/iter"
)

type Transaction struct {
	id    uuid.UUID
	sinks []gat2.Sink
}

func NewTransaction(sinks []gat2.Sink) *Transaction {
	return &Transaction{
		id:    uuid.New(),
		sinks: sinks,
	}
}

func (T *Transaction) ID() uuid.UUID {
	return T.id
}

func (T *Transaction) Route(w gat2.Work) iter.Iter[chan<- gat2.Work] {
	return iter.Flatten(
		iter.Map(
			iter.Slice(T.sinks),
			func(s gat2.Sink) iter.Iter[chan<- gat2.Work] {
				return s.Route(w)
			},
		),
	)
}

func (T *Transaction) KillSource(source gat2.Source) {
	for _, sink := range T.sinks {
		sink.KillSource(source)
	}
}

var _ gat2.Sink = (*Transaction)(nil)
