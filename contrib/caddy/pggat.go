package caddy

import (
	"github.com/caddyserver/caddy/v2"
)

func init() {
	caddy.RegisterModule((*PGGat)(nil))
}

type PGGat struct {
	Servers []Server `json:"servers,omitempty"`
}

func (*PGGat) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat",
		New: func() caddy.Module {
			return new(PGGat)
		},
	}
}

func (T *PGGat) Start() error {
	// TODO(garet)
	return nil
}

func (T *PGGat) Stop() error {
	// TODO(garet)
	return nil
}

var _ caddy.Module = (*PGGat)(nil)
var _ caddy.App = (*PGGat)(nil)
