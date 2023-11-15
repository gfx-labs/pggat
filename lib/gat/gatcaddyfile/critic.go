package gatcaddyfile

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/critics/latency"
)

func init() {
	RegisterDirective(Critic, "latency", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		if !d.NextArg() {
			return nil, d.ArgErr()
		}
		threshold, err := caddy.ParseDuration(d.Val())
		if err != nil {
			return nil, d.WrapErr(err)
		}

		return &latency.Critic{
			Threshold: caddy.Duration(threshold),
		}, nil
	})
}
