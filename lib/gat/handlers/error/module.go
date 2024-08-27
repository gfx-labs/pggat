package error_handler

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/perror"
	"github.com/caddyserver/caddy/v2"
)

func init() {
	caddy.RegisterModule((*Module)(nil))
}

type Module struct {
	Message string `json:"message"`
}

func (T *Module) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.error",
		New: func() caddy.Module {
			return new(Module)
		},
	}
}

func (T *Module) Handle(gat.Router) gat.Router {
	return gat.RouterFunc(func(ctx context.Context, c *fed.Conn) error {
		return perror.New(
			perror.FATAL,
			perror.InternalError,
			T.Message,
		)
	})
}

var _ gat.Handler = (*Module)(nil)
var _ caddy.Module = (*Module)(nil)
