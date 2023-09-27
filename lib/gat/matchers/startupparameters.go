package matchers

import (
	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/gat"
)

func init() {
	caddy.RegisterModule((*StartupParameters)(nil))
}

type StartupParameters struct {
	Parameters map[string]string `json:"startup_parameters"`
}

func (T *StartupParameters) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.matchers.startup_parameters",
		New: func() caddy.Module {
			return new(StartupParameters)
		},
	}
}

var _ gat.Matcher = (*StartupParameters)(nil)
var _ caddy.Module = (*StartupParameters)(nil)
