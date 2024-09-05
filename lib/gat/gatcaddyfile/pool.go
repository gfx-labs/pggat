package gatcaddyfile

import (
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/pools/basic"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/pools/hybrid"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

const defaultClientAcquireTimeout = caddy.Duration(time.Minute)
const defaultServerIdleTimeout = caddy.Duration(5 * time.Minute)
const defaultServerReconnectInitialTime = caddy.Duration(5 * time.Second)
const defaultServerReconnectMaxTime = caddy.Duration(1 * time.Minute)

func defaultTrackedParameters() []strutil.CIString {
	return []strutil.CIString{
		strutil.MakeCIString("client_encoding"),
		strutil.MakeCIString("datestyle"),
		strutil.MakeCIString("timezone"),
		strutil.MakeCIString("standard_conforming_strings"),
		strutil.MakeCIString("application_name"),
		strutil.MakeCIString("intervalstyle"),
		strutil.MakeCIString("search_path"),
	}
}

func defaultPoolConfig(base basic.Config) basic.Config {
	base.ClientAcquireTimeout = defaultClientAcquireTimeout
	base.ServerIdleTimeout = defaultServerIdleTimeout
	base.ServerReconnectInitialTime = defaultServerReconnectInitialTime
	base.ServerReconnectMaxTime = defaultServerReconnectMaxTime
	base.TrackedParameters = defaultTrackedParameters()
	return base
}

var defaultPool = &basic.Factory{
	Config: defaultPoolConfig(basic.Transaction),
}

func init() {
	RegisterDirective(Pool, "basic", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		module := *defaultPool

		if d.NextArg() {
			switch d.Val() {
			case "transaction":
				module.Config = defaultPoolConfig(basic.Transaction)
			case "session":
				module.Config = defaultPoolConfig(basic.Session)
			default:
				return nil, d.ArgErr()
			}
			if !d.NextBlock(d.Nesting()) {
				return &module, nil
			}
		} else {
			module.TrackedParameters = nil
		}

		for nesting := d.Nesting(); d.NextBlock(nesting); {
			directive := d.Val()
			switch directive {
			case "pooler":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				var err error
				module.RawPoolerFactory, err = UnmarshalDirectiveJSONModuleObject(
					d,
					Pooler,
					"pooler",
					warnings,
				)
				if err != nil {
					return nil, err
				}
			case "release_after_transaction":
				if d.NextArg() {
					switch d.Val() {
					case "true":
						module.ReleaseAfterTransaction = true
					case "false":
						module.ReleaseAfterTransaction = false
					default:
						return nil, d.ArgErr()
					}
				} else {
					module.ReleaseAfterTransaction = true
				}
			case "parameter_status_sync":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				module.ParameterStatusSync = basic.ParameterStatusSync(d.Val())
			case "extended_query_sync":
				if d.NextArg() {
					switch d.Val() {
					case "true":
						module.ExtendedQuerySync = true
					case "false":
						module.ExtendedQuerySync = false
					default:
						return nil, d.ArgErr()
					}
				} else {
					module.ExtendedQuerySync = true
				}
			case "packet_tracing_option":
				if d.NextArg() {
					opt, err := basic.MapTracingOption(d.Val())
					if err != nil {
						return nil, err
					}
					module.PacketTracingOption = opt
				} else {
					module.PacketTracingOption = basic.TracingOptionDisabled
				}
			case "otel_tracing_option":
				if d.NextArg() {
					opt, err := basic.MapTracingOption(d.Val())
					if err != nil {
						return nil, err
					}
					module.OtelTracingOption = opt
				} else {
					module.OtelTracingOption = basic.TracingOptionDisabled
				}
			case "reset_query":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				module.ServerResetQuery = d.Val()
			case "idle_timeout":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				val, err := time.ParseDuration(d.Val())
				if err != nil {
					return nil, d.WrapErr(err)
				}

				module.ServerIdleTimeout = caddy.Duration(val)
			case "reconnect":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				initialTime, err := time.ParseDuration(d.Val())
				if err != nil {
					return nil, d.WrapErr(err)
				}

				maxTime := initialTime
				if d.NextArg() {
					maxTime, err = time.ParseDuration(d.Val())
					if err != nil {
						return nil, d.WrapErr(err)
					}
				}

				module.ServerReconnectInitialTime = caddy.Duration(initialTime)
				module.ServerReconnectMaxTime = caddy.Duration(maxTime)
			case "track":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				critic, err := UnmarshalDirectiveJSONModuleObject(
					d,
					Critic,
					"critic",
					warnings,
				)
				if err != nil {
					return nil, err
				}

				module.RawCritics = append(module.RawCritics, critic)
			default:
				return nil, d.ArgErr()
			}
		}

		return &module, nil
	})

	RegisterDirective(Pool, "hybrid", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		module := hybrid.Factory{
			Config: hybrid.Config{
				ClientAcquireTimeout:       defaultClientAcquireTimeout,
				ServerIdleTimeout:          defaultServerIdleTimeout,
				ServerReconnectInitialTime: defaultServerReconnectInitialTime,
				ServerReconnectMaxTime:     defaultServerReconnectMaxTime,
				TrackedParameters:          defaultTrackedParameters(),
			},
		}

		module.TrackedParameters = nil

		for nesting := d.Nesting(); d.NextBlock(nesting); {
			directive := d.Val()
			switch directive {
			case "idle_timeout":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				val, err := time.ParseDuration(d.Val())
				if err != nil {
					return nil, d.WrapErr(err)
				}

				module.ServerIdleTimeout = caddy.Duration(val)
			case "reconnect":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				initialTime, err := time.ParseDuration(d.Val())
				if err != nil {
					return nil, d.WrapErr(err)
				}

				maxTime := initialTime
				if d.NextArg() {
					maxTime, err = time.ParseDuration(d.Val())
					if err != nil {
						return nil, d.WrapErr(err)
					}
				}

				module.ServerReconnectInitialTime = caddy.Duration(initialTime)
				module.ServerReconnectMaxTime = caddy.Duration(maxTime)
			case "track":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				module.TrackedParameters = append(module.TrackedParameters, strutil.MakeCIString(d.Val()))
			case "penalize":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				critic, err := UnmarshalDirectiveJSONModuleObject(
					d,
					Critic,
					"critic",
					warnings,
				)
				if err != nil {
					return nil, err
				}

				module.RawCritics = append(module.RawCritics, critic)
			default:
				return nil, d.ArgErr()
			}
		}

		return &module, nil
	})
}
