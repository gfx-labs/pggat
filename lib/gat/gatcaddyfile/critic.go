package gatcaddyfile

import (
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/critics/latency"
)

func init() {
	RegisterDirective(Critic, "latency", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		module := &latency.Critic{
			Validity: caddy.Duration(5 * time.Minute),
		}

		if !d.NextArg() {
			return nil, d.ArgErr()
		}

		threshold, err := caddy.ParseDuration(d.Val())
		if err != nil {
			return nil, d.WrapErr(err)
		}
		module.Threshold = caddy.Duration(threshold)

		if d.NextArg() {
			// optional validity
			var validity time.Duration
			validity, err = caddy.ParseDuration(d.Val())
			if err != nil {
				return nil, d.WrapErr(err)
			}
			module.Validity = caddy.Duration(validity)
		}

		return module, nil
	})
}
