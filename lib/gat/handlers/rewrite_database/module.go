package rewrite_database

import (
	"errors"
	"strings"

	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat"
)

func init() {
	caddy.RegisterModule((*Module)(nil))
}

type Module struct {
	Mode     string `json:"mode,omitempty"`
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

func (T *Module) Validate() error {
	switch T.Mode {
	case "strip_prefix", "strip_suffix", "":
		return nil
	default:
		return errors.New("unknown rewrite mode")
	}
}

func (T *Module) Handle(next gat.Router) gat.Router {
	return gat.RouterFunc(func(conn *fed.Conn) error {
		switch T.Mode {
		case "strip_prefix":
			conn.Database = strings.TrimPrefix(conn.Database, T.Database)
		case "strip_suffix":
			conn.Database = strings.TrimSuffix(conn.Database, T.Database)
		default:
			conn.Database = T.Database
		}
		return next.Route(conn)
	})
}

var _ gat.Handler = (*Module)(nil)
var _ caddy.Module = (*Module)(nil)
var _ caddy.Validator = (*Module)(nil)
