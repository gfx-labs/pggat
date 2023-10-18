package gatcaddyfile

import (
	"strconv"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	error_handler "gfx.cafe/gfx/pggat/lib/gat/handlers/error"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pgbouncer_spilo"
	pool_handler "gfx.cafe/gfx/pggat/lib/gat/handlers/pool"

	"gfx.cafe/gfx/pggat/lib/bouncer"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/allowed_startup_parameters"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/discovery"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pgbouncer"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/require_ssl"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/rewrite_database"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/rewrite_parameter"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/rewrite_password"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/rewrite_user"
	"gfx.cafe/gfx/pggat/lib/gat/ssl/clients/insecure_skip_verify"
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

		mode := d.Val()

		if d.NextArg() {
			return &rewrite_user.Module{
				Mode: mode,
				User: d.Val(),
			}, nil
		} else {
			return &rewrite_user.Module{
				User: mode,
			}, nil
		}
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

		mode := d.Val()

		if d.NextArg() {
			return &rewrite_database.Module{
				Mode:     mode,
				Database: d.Val(),
			}, nil
		} else {
			return &rewrite_database.Module{
				Database: mode,
			}, nil
		}
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
	RegisterDirective(Handler, "error", func(d *caddyfile.Dispenser, _ *[]caddyconfig.Warning) (caddy.Module, error) {
		if !d.NextArg() {
			return nil, d.ArgErr()
		}

		message := d.Val()

		return &error_handler.Module{
			Message: message,
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
				ReconcilePeriod: caddy.Duration(5 * time.Minute),
				Pool: JSONModuleObject(
					defaultPool,
					Pool,
					"pool",
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
					module.ReconcilePeriod = caddy.Duration(val)
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
				case "pool":
					if !d.NextArg() {
						return nil, d.ArgErr()
					}

					var err error
					module.Pool, err = UnmarshalDirectiveJSONModuleObject(
						d,
						Pool,
						"pool",
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
	RegisterDirective(Handler, "pgbouncer_spilo", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		module := pgbouncer_spilo.Module{}

		if !d.NextBlock(d.Nesting()) {
			return nil, d.ArgErr()
		}

		for {
			if d.Val() == "}" {
				break
			}

			directive := d.Val()
			switch directive {
			case "host":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				module.Host = d.Val()
			case "port":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				var err error
				module.Port, err = strconv.Atoi(d.Val())
				if err != nil {
					return nil, d.WrapErr(err)
				}
			case "user":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				module.User = d.Val()
			case "schema":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				module.Schema = d.Val()
			case "password":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				module.Password = d.Val()
			case "mode":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				module.Mode = d.Val()
			case "default_size":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				var err error
				module.DefaultSize, err = strconv.Atoi(d.Val())
				if err != nil {
					return nil, d.WrapErr(err)
				}
			case "min_size":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				var err error
				module.MinSize, err = strconv.Atoi(d.Val())
				if err != nil {
					return nil, d.WrapErr(err)
				}
			case "reserve_size":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				var err error
				module.ReserveSize, err = strconv.Atoi(d.Val())
				if err != nil {
					return nil, d.WrapErr(err)
				}
			case "max_client_conn":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				var err error
				module.MaxClientConn, err = strconv.Atoi(d.Val())
				if err != nil {
					return nil, d.WrapErr(err)
				}
			case "max_db_conn":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				var err error
				module.MaxDBConn, err = strconv.Atoi(d.Val())
				if err != nil {
					return nil, d.WrapErr(err)
				}
			default:
				return nil, d.ArgErr()
			}

			if !d.NextLine() {
				return nil, d.EOFErr()
			}
		}

		return &module, nil
	})
	RegisterDirective(Handler, "pool", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		module := pool_handler.Module{
			Pool: JSONModuleObject(
				defaultPool,
				Pool,
				"pool",
				warnings,
			),
			Recipe: pool_handler.Recipe{
				Dialer: pool_handler.Dialer{
					SSLMode: bouncer.SSLModePrefer,
					RawSSL: JSONModuleObject(
						&insecure_skip_verify.Client{},
						SSLClient,
						"provider",
						warnings,
					),
				},
			},
		}

		if d.NextArg() {
			module.Recipe.Dialer.Address = d.Val()

			if !d.NextArg() {
				return nil, d.ArgErr()
			}
			module.Recipe.Dialer.Database = d.Val()

			if !d.NextArg() {
				return nil, d.ArgErr()
			}
			module.Recipe.Dialer.Username = d.Val()

			if !d.NextArg() {
				return nil, d.ArgErr()
			}
			module.Recipe.Dialer.RawPassword = d.Val()
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
				case "pool":
					if !d.NextArg() {
						return nil, d.ArgErr()
					}

					var err error
					module.Pool, err = UnmarshalDirectiveJSONModuleObject(
						d,
						Pool,
						"pool",
						warnings,
					)
					if err != nil {
						return nil, err
					}
				case "address":
					if !d.NextArg() {
						return nil, d.ArgErr()
					}

					module.Recipe.Dialer.Address = d.Val()
				case "ssl":
					if !d.NextArg() {
						return nil, d.ArgErr()
					}

					module.Recipe.Dialer.SSLMode = bouncer.SSLMode(d.Val())

					if !d.NextArg() {
						return nil, d.ArgErr()
					}

					var err error
					module.Recipe.Dialer.RawSSL, err = UnmarshalDirectiveJSONModuleObject(
						d,
						SSLClient,
						"provider",
						warnings,
					)
					if err != nil {
						return nil, err
					}
				case "username":
					if !d.NextArg() {
						return nil, d.ArgErr()
					}

					module.Recipe.Dialer.Username = d.Val()
				case "password":
					if !d.NextArg() {
						return nil, d.ArgErr()
					}

					module.Recipe.Dialer.RawPassword = d.Val()
				case "database":
					if !d.NextArg() {
						return nil, d.ArgErr()
					}

					module.Recipe.Dialer.Database = d.Val()
				case "parameter":
					if !d.NextArg() {
						return nil, d.ArgErr()
					}

					keyValue := d.Val()
					key, value, ok := strings.Cut(keyValue, "=")
					if !ok {
						return nil, d.SyntaxErr("key=value")
					}
					if module.Recipe.Dialer.RawParameters == nil {
						module.Recipe.Dialer.RawParameters = make(map[string]string)
					}
					module.Recipe.Dialer.RawParameters[key] = value
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
}
