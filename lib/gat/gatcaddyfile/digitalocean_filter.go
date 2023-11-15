package gatcaddyfile

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/discovery/discoverers/digitalocean/filters/tag"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func init() {
	RegisterDirective(
		DigitaloceanFilter,
		"tag",
		func(d *caddyfile.Dispenser, _ *[]caddyconfig.Warning) (caddy.Module, error) {
			if !d.NextArg() {
				return nil, d.ArgErr()
			}

			return &tag.Filter{
				Tag: strutil.Matcher(d.Val()),
			}, nil
		},
	)
}
