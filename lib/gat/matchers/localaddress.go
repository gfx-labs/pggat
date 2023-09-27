package matchers

import (
	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/gat"
)

func init() {
	caddy.RegisterModule((*LocalAddress)(nil))
}

type LocalAddress struct {
	Address string `json:"address"`
}

func (T *LocalAddress) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.matchers.local_address",
		New: func() caddy.Module {
			return new(LocalAddress)
		},
	}
}

var _ gat.Matcher = (*LocalAddress)(nil)
var _ caddy.Module = (*LocalAddress)(nil)
