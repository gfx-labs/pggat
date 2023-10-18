package gatcaddyfile

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/poolers/lifo"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/poolers/rob"
)

func init() {
	RegisterDirective(Pooler, "lifo", func(d *caddyfile.Dispenser, _ *[]caddyconfig.Warning) (caddy.Module, error) {
		return &lifo.Factory{}, nil
	})
	RegisterDirective(Pooler, "rob", func(d *caddyfile.Dispenser, _ *[]caddyconfig.Warning) (caddy.Module, error) {
		return &rob.Factory{}, nil
	})
}
