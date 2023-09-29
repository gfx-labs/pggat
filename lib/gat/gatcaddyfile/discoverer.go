package gatcaddyfile

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/discovery/discoverers/digitalocean"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/discovery/discoverers/google_cloud_sql"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/discovery/discoverers/zalando_operator"
)

func init() {
	RegisterDirective(Discoverer, "digitalocean", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		if !d.NextArg() {
			return nil, d.ArgErr()
		}

		apiKey := d.Val()

		return &digitalocean.Discoverer{
			Config: digitalocean.Config{
				APIKey: apiKey,
			},
		}, nil
	})
	RegisterDirective(Discoverer, "google_cloud_sql", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		module := google_cloud_sql.Discoverer{
			Config: google_cloud_sql.Config{
				IpAddressType: "PRIMARY",
			},
		}

		if !d.NextArg() {
			return nil, d.ArgErr()
		}
		module.Project = d.Val()

		if !d.NextArg() {
			return nil, d.ArgErr()
		}
		module.AuthUser = d.Val()

		if !d.NextArg() {
			return nil, d.ArgErr()
		}
		module.AuthPassword = d.Val()

		return &module, nil
	})
	RegisterDirective(Discoverer, "zalando_operator", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		module := zalando_operator.Discoverer{
			Config: zalando_operator.Config{
				Namespace: "default",
			},
		}

		if !d.NextArg() {
			return nil, d.ArgErr()
		}
		module.OperatorConfigurationObject = d.Val()

		return &module, nil
	})
}
