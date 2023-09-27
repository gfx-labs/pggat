package matchers

import (
	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat"
)

func init() {
	caddy.RegisterModule((*Network)(nil))
}

type Network struct {
	Network string `json:"network"`
}

func (T *Network) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.matchers.network",
		New: func() caddy.Module {
			return new(Network)
		},
	}
}

func (T *Network) Matches(conn fed.Conn) bool {
	return conn.LocalAddr().Network() == T.Network
}

var _ gat.Matcher = (*Network)(nil)
var _ caddy.Module = (*Network)(nil)
