package gatcaddyfile

import (
	"strconv"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/discovery/discoverers/digitalocean"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/discovery/discoverers/google_cloud_sql"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/discovery/discoverers/zalando_operator"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func init() {
	RegisterDirective(Discoverer, "digitalocean", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		module := digitalocean.Discoverer{}

		if d.NextArg() {
			module.APIKey = d.Val()
		} else {
			if !d.NextBlock(d.Nesting()) {
				return nil, d.ArgErr()
			}

			for {
				if d.Val() == "}" {
					break
				}

				directive := d.Val()
				switch directive {
				case "token":
					if !d.NextArg() {
						return nil, d.ArgErr()
					}

					module.APIKey = d.Val()
				case "private":
					if d.NextArg() {
						switch d.Val() {
						case "true":
							module.Private = true
						case "false":
							module.Private = false
						default:
							return nil, d.ArgErr()
						}
					} else {
						module.Private = true
					}
				case "filter":
					if !d.NextArg() {
						return nil, d.ArgErr()
					}

					module.Filter = strutil.Matcher(d.Val())
				case "priority":
					if !d.NextArg() {
						return nil, d.ArgErr()
					}

					keyValue := d.Val()
					filter, value, ok := strings.Cut(keyValue, "=")
					priority := int64(-1)
					if ok {
						var err error
						priority, err = strconv.ParseInt(value, 10, 64)
						if err != nil {
							return nil, d.WrapErr(err)
						}
					}

					module.Priority = append(module.Priority, digitalocean.Priority{
						Filter: strutil.Matcher(filter),
						Value:  int(priority),
					})
				default:
					return nil, d.ArgErr()
				}

				if !d.NextLine() {
					return nil, d.EOFErr()
				}
			}
		}

		return &module, nil
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
