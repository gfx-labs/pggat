package rewrite_user

import (
	"context"
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
	Mode string `json:"mode,omitempty"`
	User string `json:"user"`
}

func (T *Module) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.rewrite_user",
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
	return gat.RouterFunc(func(ctx context.Context, conn *fed.Conn) error {
		switch T.Mode {
		case "strip_prefix":
			conn.User = strings.TrimPrefix(conn.User, T.User)
		case "strip_suffix":
			conn.User = strings.TrimSuffix(conn.User, T.User)
		default:
			conn.User = T.User
		}
		return next.Route(ctx, conn)
	})
}

var _ gat.Handler = (*Module)(nil)
var _ caddy.Module = (*Module)(nil)
var _ caddy.Validator = (*Module)(nil)
