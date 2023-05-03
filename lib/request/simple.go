package request

import "pggat2/lib/pnet/packet"

type Simple struct {
	query packet.Raw
}

func NewSimple(query packet.Raw) *Simple {
	if query.Type != packet.Query {
		panic("expected packet.Query")
	}
	return &Simple{
		query: query,
	}
}

func (T *Simple) Query() packet.Raw {
	return T.query
}

func (T *Simple) request() {}

var _ Request = (*Simple)(nil)
