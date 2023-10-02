package rewrite_parameter

import (
	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func init() {
	caddy.RegisterModule((*Module)(nil))
}

type Module struct {
	Key   strutil.CIString `json:"key"`
	Value string           `json:"value"`
}

func (T *Module) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.rewrite_parameter",
		New: func() caddy.Module {
			return new(Module)
		},
	}
}

func (T *Module) Handle(conn *fed.Conn) error {
	if conn.InitialParameters == nil {
		conn.InitialParameters = make(map[strutil.CIString]string)
	}
	conn.InitialParameters[T.Key] = T.Value

	return nil
}

var _ gat.Handler = (*Module)(nil)
var _ caddy.Module = (*Module)(nil)
