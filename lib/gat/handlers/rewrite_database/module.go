package rewrite_database

import (
	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat"
)

func init() {
	caddy.RegisterModule((*Module)(nil))
}

type Module struct {
	Database string `json:"database"`
}

func (T *Module) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.rewrite_database",
		New: func() caddy.Module {
			return new(Module)
		},
	}
}

func (T *Module) Handle(conn *fed.Conn) error {
	conn.Database = T.Database

	return nil
}

var _ gat.Handler = (*Module)(nil)
var _ caddy.Module = (*Module)(nil)
