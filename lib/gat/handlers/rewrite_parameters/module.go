package rewrite_parameters

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
	Parameters map[string]string `json:"parameters"`

	parameters map[strutil.CIString]string
}

func (T *Module) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.rewrite_parameters",
		New: func() caddy.Module {
			return new(Module)
		},
	}
}

func (T *Module) Provision(ctx caddy.Context) error {
	T.parameters = make(map[strutil.CIString]string, len(T.Parameters))

	for key, value := range T.Parameters {
		T.parameters[strutil.MakeCIString(key)] = value
	}

	return nil
}

func (T *Module) Handle(conn *fed.Conn) error {
	if conn.InitialParameters == nil {
		conn.InitialParameters = make(map[strutil.CIString]string)
	}

	for key, value := range T.parameters {
		conn.InitialParameters[key] = value
	}

	return nil
}

var _ gat.Handler = (*Module)(nil)
var _ caddy.Module = (*Module)(nil)
var _ caddy.Provisioner = (*Module)(nil)
