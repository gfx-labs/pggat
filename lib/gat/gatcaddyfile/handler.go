package gatcaddyfile

import (
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	"gfx.cafe/gfx/pggat/lib/bouncer"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/allowed_startup_parameters"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/discovery"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pgbouncer"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/require_ssl"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/rewrite_database"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/rewrite_parameter"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/rewrite_password"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/rewrite_user"
	"gfx.cafe/gfx/pggat/lib/gat/poolers/transaction"
	"gfx.cafe/gfx/pggat/lib/gat/ssl/clients/insecure_skip_verify"
	"gfx.cafe/gfx/pggat/lib/util/dur"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func init() {
	RegisterDirective(Handler, "require_ssl", func(d *caddyfile.Dispenser, _ *[]caddyconfig.Warning) (caddy.Module, error) {
		var ssl = true
		if d.NextArg() {
			switch d.Val() {
			case "true":
				ssl = true
			case "false":
				ssl = false
			default:
				return nil, d.SyntaxErr("boolean")
			}
		}
		return &require_ssl.Module{
			SSL: ssl,
		}, nil
	})
	RegisterDirective(Handler, "allowed_startup_parameters", func(d *caddyfile.Dispenser, _ *[]caddyconfig.Warning) (caddy.Module, error) {
		if !d.NextBlock(d.Nesting()) {
			return nil, d.ArgErr()
		}

		var module allowed_startup_parameters.Module

		for {
			if d.Val() == "}" {
				break
			}

			module.Parameters = append(module.Parameters, strutil.MakeCIString(d.Val()))

			if !d.NextLine() {
				return nil, d.EOFErr()
			}
		}

		return &module, nil
	})
	RegisterDirective(Handler, "user", func(d *caddyfile.Dispenser, _ *[]caddyconfig.Warning) (caddy.Module, error) {
		if !d.NextArg() {
			return nil, d.ArgErr()
		}

		return &rewrite_user.Module{
			User: d.Val(),
		}, nil
	})
	RegisterDirective(Handler, "password", func(d *caddyfile.Dispenser, _ *[]caddyconfig.Warning) (caddy.Module, error) {
		if !d.NextArg() {
			return nil, d.ArgErr()
		}

		return &rewrite_password.Module{
			Password: d.Val(),
		}, nil
	})
	RegisterDirective(Handler, "database", func(d *caddyfile.Dispenser, _ *[]caddyconfig.Warning) (caddy.Module, error) {
		if !d.NextArg() {
			return nil, d.ArgErr()
		}

		return &rewrite_database.Module{
			Database: d.Val(),
		}, nil
	})
	RegisterDirective(Handler, "parameter", func(d *caddyfile.Dispenser, _ *[]caddyconfig.Warning) (caddy.Module, error) {
		if !d.NextArg() {
			return nil, d.ArgErr()
		}

		keyValue := d.Val()
		key, value, ok := strings.Cut(keyValue, "=")
		if !ok {
			return nil, d.SyntaxErr("key=value")
		}

		return &rewrite_parameter.Module{
			Key:   strutil.MakeCIString(key),
			Value: value,
		}, nil
	})
	RegisterDirective(Handler, "pgbouncer", func(d *caddyfile.Dispenser, _ *[]caddyconfig.Warning) (caddy.Module, error) {
		var config = "pgbouncer.ini"
		if d.NextArg() {
			config = d.Val()
		}
		return &pgbouncer.Module{
			ConfigFile: config,
		}, nil
	})
	RegisterDirective(Handler, "discovery", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		module := discovery.Module{
			Config: discovery.Config{
				ReconcilePeriod: dur.Duration(5 * time.Minute),
				Pooler: JSONModuleObject(
					&transaction.Pool{
						ManagementConfig: defaultPoolManagementConfig,
					},
					Pooler,
					"pooler",
					warnings,
				),
				ServerSSLMode: bouncer.SSLModePrefer,
				ServerSSL: JSONModuleObject(
					&insecure_skip_verify.Client{},
					SSLClient,
					"provider",
					warnings,
				),
			},
		}

		if d.NextArg() {
			// discoverer
			var err error
			module.Discoverer, err = UnmarshalDirectiveJSONModuleObject(
				d,
				Discoverer,
				"discoverer",
				warnings,
			)

			if err != nil {
				return nil, err
			}
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
				case "reconcile_period":
					if !d.NextArg() {
						return nil, d.ArgErr()
					}

					val, err := time.ParseDuration(d.Val())
					if err != nil {
						return nil, d.WrapErr(err)
					}
					module.ReconcilePeriod = dur.Duration(val)
				case "discoverer":
					if !d.NextArg() {
						return nil, d.ArgErr()
					}

					var err error
					module.Discoverer, err = UnmarshalDirectiveJSONModuleObject(
						d,
						Discoverer,
						"discoverer",
						warnings,
					)
					if err != nil {
						return nil, err
					}
				case "pooler":
					if !d.NextArg() {
						return nil, d.ArgErr()
					}

					var err error
					module.Pooler, err = UnmarshalDirectiveJSONModuleObject(
						d,
						Pooler,
						"pooler",
						warnings,
					)
					if err != nil {
						return nil, err
					}
				case "ssl":
					if !d.NextArg() {
						return nil, d.ArgErr()
					}

					module.ServerSSLMode = bouncer.SSLMode(d.Val())

					if !d.NextArg() {
						return nil, d.ArgErr()
					}

					var err error
					module.ServerSSL, err = UnmarshalDirectiveJSONModuleObject(
						d,
						SSLClient,
						"provider",
						warnings,
					)
					if err != nil {
						return nil, err
					}
				case "parameter":
					if !d.NextArg() {
						return nil, d.ArgErr()
					}

					keyValue := d.Val()
					key, value, ok := strings.Cut(keyValue, "=")
					if !ok {
						return nil, d.SyntaxErr("key=value")
					}
					if module.ServerStartupParameters == nil {
						module.ServerStartupParameters = make(map[string]string)
					}
					module.ServerStartupParameters[key] = value
				}

				if !d.NextLine() {
					return nil, d.EOFErr()
				}
			}
		}

		return &module, nil
	})
}
