package session

import (
	"pggat2/lib/gat"
	"pggat2/lib/zap"
)

type Pool struct {
}

func (T *Pool) Serve(client zap.ReadWriter) {
	// TODO implement me
	panic("implement me")
}

var _ gat.Pool = (*Pool)(nil)
