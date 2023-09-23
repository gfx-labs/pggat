package discovery

import (
	"pggat/lib/gat"
	"pggat/lib/gat/metrics"
)

type Module struct {
}

func NewModule(config Config) (*Module, error) {

}

func (T *Module) GatModule() gat.ModuleInfo {
	// TODO(garet)
}

func (T *Module) ReadMetrics(metrics *metrics.Pools) {
	// TODO implement me
	panic("implement me")
}

func (T *Module) Lookup(user, database string) *gat.Pool {
	// TODO implement me
	panic("implement me")
}

var _ gat.Module = (*Module)(nil)
var _ gat.Provider = (*Module)(nil)
