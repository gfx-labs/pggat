package rewrite_password

import (
	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/auth/credentials"
	"gfx.cafe/gfx/pggat/lib/bouncer/frontends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat"
)

func init() {
	caddy.RegisterModule((*Module)(nil))
}

type Module struct {
	Password string `json:"password"`
}

func (T *Module) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.rewrite_password",
		New: func() caddy.Module {
			return new(Module)
		},
	}
}

func (T *Module) Handle(next gat.Router) gat.Router {
	return gat.RouterFunc(func(conn *fed.Conn) error {
		if err := frontends.Authenticate(
			conn,
			credentials.FromString(conn.User, T.Password),
		); err != nil {
			return err
		}

		return next.Route(conn)
	})
}

var _ gat.Handler = (*Module)(nil)
var _ caddy.Module = (*Module)(nil)
