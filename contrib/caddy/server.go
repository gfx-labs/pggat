package caddy

import (
	"github.com/caddyserver/caddy/v2"
)

func init() {
	caddy.RegisterModule(Server{})
}

type Server struct{}

func (T Server) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "postgres.poolers.pggat",
		New: func() caddy.Module {
			return Server{}
		},
	}
}

var _ caddy.Module = Server{}
