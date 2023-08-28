package session

import "pggat2/lib/gat"

func NewPool(options gat.PoolOptions) *gat.Pool {
	options.Pooler = new(Pooler)
	return gat.NewPool(options)
}
