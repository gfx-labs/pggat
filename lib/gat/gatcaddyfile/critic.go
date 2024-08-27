package gatcaddyfile

import (
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/critics/replication"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/critics/latency"
)

func init() {
	// Register a directive for the latency critic which measures query response
	// time as the determining factor for load balancing between replicas
	RegisterDirective(Critic, "latency", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		module := &latency.Critic{
			Validity: caddy.Duration(5 * time.Minute),
		}

		// parse nested format
		if d.NextBlock(d.Nesting()) {
			return parseLatency(module, d, warnings)
		}

		// parse legacy format
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

	// Register a directive handler for the replication critic which uses
	// replication lag as a determining factor for load balancing between replicas
	RegisterDirective(Critic, "replication", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		module := &replication.Critic{
			Validity: caddy.Duration(2 * time.Minute),
		}

		if !d.NextBlock(d.Nesting()) {
			return nil, d.ArgErr()
		}

		for {
			if d.Val() == "}" {
				break
			}

			directive := d.Val()
			switch directive {
			case "validity":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				validity, err := time.ParseDuration(d.Val())
				if err != nil {
					return nil, d.WrapErr(err)
				}

				module.Validity = caddy.Duration(validity)
			case "threshold":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				threshold, err := time.ParseDuration(d.Val())
				if err != nil {
					return nil, d.WrapErr(err)
				}

				module.Threshold = caddy.Duration(threshold)
			default:
				return nil, d.ArgErr()
			}

			if !d.NextLine() {
				return nil, d.EOFErr()
			}
		}

		return module, nil
	})
}

func parseLatency(critic *latency.Critic, d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
	for {
		if d.Val() == "}" {
			break
		}

		directive := d.Val()
		switch directive {
		case "validity":
			if !d.NextArg() {
				return nil, d.ArgErr()
			}

			validity, err := time.ParseDuration(d.Val())
			if err != nil {
				return nil, d.WrapErr(err)
			}

			critic.Validity = caddy.Duration(validity)
		case "threshold":
			if !d.NextArg() {
				return nil, d.ArgErr()
			}

			threshold, err := time.ParseDuration(d.Val())
			if err != nil {
				return nil, d.WrapErr(err)
			}

			critic.Threshold = caddy.Duration(threshold)
		default:
			return nil, d.ArgErr()
		}

		if !d.NextLine() {
			return nil, d.EOFErr()
		}
	}

	return critic, nil
}
