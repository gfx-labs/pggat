package require_ssl

import (
	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/perror"
)

func init() {
	caddy.RegisterModule((*Module)(nil))
}

type Module struct {
	SSL bool `json:"ssl"`
}

func (T *Module) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.require_ssl",
		New: func() caddy.Module {
			return new(Module)
		},
	}
}

func (T *Module) Handle(next gat.Router) gat.Router {
	return gat.RouterFunc(func(conn *fed.Conn) error {
		if T.SSL {
			if !conn.SSL {
				return perror.New(
					perror.FATAL,
					perror.InvalidPassword,
					"SSL is required",
				)
			}
			return next.Route(conn)
		}

		if conn.SSL {
			return perror.New(
				perror.FATAL,
				perror.InvalidPassword,
				"SSL is not allowed",
			)
		}
		return next.Route(conn)
	})
}

var _ gat.Handler = (*Module)(nil)
var _ caddy.Module = (*Module)(nil)
