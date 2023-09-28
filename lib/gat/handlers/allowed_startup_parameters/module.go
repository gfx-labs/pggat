package module

import (
	"fmt"

	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/perror"
	"gfx.cafe/gfx/pggat/lib/util/slices"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func init() {
	caddy.RegisterModule((*Module)(nil))
}

type Module struct {
	Parameters []strutil.CIString `json:"parameters"`
}

func (T *Module) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.allowed_startup_parameters",
		New: func() caddy.Module {
			return new(Module)
		},
	}
}

func (T *Module) Handle(conn *fed.Conn) error {
	for parameter := range conn.InitialParameters {
		if !slices.Contains(T.Parameters, parameter) {
			return perror.New(
				perror.FATAL,
				perror.FeatureNotSupported,
				fmt.Sprintf(`Startup parameter "%s" is not supported`, parameter.String()),
			)
		}
	}

	return nil
}

var _ gat.Handler = (*Module)(nil)
var _ caddy.Module = (*Module)(nil)
