package source

import (
	"github.com/google/uuid"

	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v2/pool"
)

type Source struct {
	uuid uuid.UUID
	pool *pool.Pool
}

func NewSource(p *pool.Pool) *Source {
	return &Source{
		uuid: uuid.New(),
		pool: p,
	}
}

func (T *Source) Schedule(work any, constraints rob.Constraints) {
	T.pool.Schedule(T.uuid, work, constraints)
}

var _ rob.Source = (*Source)(nil)
