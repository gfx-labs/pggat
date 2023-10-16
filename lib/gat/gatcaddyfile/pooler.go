package gatcaddyfile

import (
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	"gfx.cafe/gfx/pggat/lib/gat/pool"
	"gfx.cafe/gfx/pggat/lib/util/dur"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

var defaultPoolManagementConfig = pool.ManagementConfig{
	ServerIdleTimeout:          dur.Duration(5 * time.Minute),
	ServerReconnectInitialTime: dur.Duration(5 * time.Second),
	ServerReconnectMaxTime:     dur.Duration(1 * time.Minute),
	TrackedParameters: []strutil.CIString{
		strutil.MakeCIString("client_encoding"),
		strutil.MakeCIString("datestyle"),
		strutil.MakeCIString("timezone"),
		strutil.MakeCIString("standard_conforming_strings"),
		strutil.MakeCIString("application_name"),
	},
}

func unmarshalPoolConfig(d *caddyfile.Dispenser) (pool.ManagementConfig, error) {
	var config = defaultPoolManagementConfig

	if !d.NextBlock(d.Nesting()) {
		return config, nil
	}

	config.TrackedParameters = nil

	for {
		if d.Val() == "}" {
			break
		}

		directive := d.Val()
		switch directive {
		case "idle_timeout":
			if !d.NextArg() {
				return config, d.ArgErr()
			}

			val, err := time.ParseDuration(d.Val())
			if err != nil {
				return config, d.WrapErr(err)
			}
			config.ServerIdleTimeout = dur.Duration(val)
		case "reconnect":
			if !d.NextArg() {
				return config, d.ArgErr()
			}

			initialTime, err := time.ParseDuration(d.Val())
			if err != nil {
				return config, d.WrapErr(err)
			}

			maxTime := initialTime
			if d.NextArg() {
				maxTime, err = time.ParseDuration(d.Val())
				if err != nil {
					return config, d.WrapErr(err)
				}
			}

			config.ServerReconnectInitialTime = dur.Duration(initialTime)
			config.ServerReconnectMaxTime = dur.Duration(maxTime)
		case "track":
			if !d.NextArg() {
				return config, d.ArgErr()
			}

			config.TrackedParameters = append(config.TrackedParameters, strutil.MakeCIString(d.Val()))
		default:
			return config, d.ArgErr()
		}

		if !d.NextLine() {
			return config, d.EOFErr()
		}
	}

	return config, nil
}

func init() {
	RegisterDirective(Pooler, "transaction", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		config, err := unmarshalPoolConfig(d)
		if err != nil {
			return nil, err
		}

		return &rob.Factory{
			ManagementConfig: config,
		}, nil
	})
	RegisterDirective(Pooler, "session", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		config, err := unmarshalPoolConfig(d)
		if err != nil {
			return nil, err
		}

		return &lifo.Factory{
			ManagementConfig: config,
		}, nil
	})
}
