package metrics

import (
	"github.com/google/uuid"

	"pggat2/lib/util/maps"
)

type Pool struct {
	Servers map[uuid.UUID]Conn
	Clients map[uuid.UUID]Conn
}

func (T *Pool) Clear() {
	maps.Clear(T.Servers)
	maps.Clear(T.Clients)
}

func (T *Pool) String() string {
	return "TODO(garet)" // TODO(garet)
}
