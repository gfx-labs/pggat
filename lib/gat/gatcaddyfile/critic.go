package gatcaddyfile

import (
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/critics/replication"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/critics/latency"
)

func init() {
	// Register a directive for the latency critic which measures query response
	// time as the determining factor for load balancing between replicas
	//
	// Config Format
	//
	//	* All fields are optional and will fall back to a suitable default
	//	* Duration values use caddy.Duration syntax
	//
	//	penalize query_latency [query_threshold duration [validity duration]]
	//
	//	-or-
	//
	//	penalize query_latency {
	//		[query_threshold] {duration}
	//		[validity] {duration}
	//	}
	//
	//	pool basic session {
	//			penalize query_latency						# use the defaults
	//			penalize query_latency 500ms			# set query threshold w/ default validity
	//			penalize query_latency 500ms	3m	# set query threshold and validity
	//			penalize query_latency {
	//				query_threshold 300ms
	//				validity 5m
	//			}
	//	}
	RegisterDirective(Critic, "query_latency", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		return parseQueryCritic(d, warnings)
	})

	// legacy directive for query_latency
	RegisterDirective(Critic, "latency", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		return parseQueryCritic(d, warnings)
	})

	// Register a directive handler for the replication critic which uses
	// replication lag as a determining factor for load balancing between
	// replicas
	//
	//	Config format
	//
	//	* All fields are optional and will fall back to a suitable default
	//	* Duration values use caddy.Duration syntax
	//
	//	pool basic session {
	//			penalize replication_latency		# valid declaration. Use default values
	//
	//			penalize replication_latency {
	//				replication_threshold 3s
	//				query_threshold 300ms
	//				validity 5m
	//			}
	//	}
	RegisterDirective(Critic, "replication_latency", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		module := replication.NewCritic()

		if !d.NextBlock(d.Nesting()) {
			return module, nil
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

				validity, err := caddy.ParseDuration(d.Val())
				if err != nil {
					return nil, d.WrapErr(err)
				}

				module.Validity = caddy.Duration(validity)
			case "replication_threshold":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				threshold, err := caddy.ParseDuration(d.Val())
				if err != nil {
					return nil, d.WrapErr(err)
				}

				module.ReplicationThreshold = caddy.Duration(threshold)
			case "query_threshold":
				if !d.NextArg() {
					return nil, d.ArgErr()
				}

				threshold, err := caddy.ParseDuration(d.Val())
				if err != nil {
					return nil, d.WrapErr(err)
				}

				module.QueryThreshold = caddy.Duration(threshold)
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

func parseQueryCritic(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
	module := latency.NewCritic()

	// parse block format
	if d.NextBlock(d.Nesting()) {
		return parseLatency(module, d, warnings)
	}

	// use the defaults
	if !d.NextArg() {
		return module, nil
	}

	// parse legacy format
	dur, err := caddy.ParseDuration(d.Val())
	if err != nil {
		return nil, d.WrapErr(err)
	}
	module.QueryThreshold = caddy.Duration(dur)

	if d.NextArg() {
		// optional validity
		dur, err = caddy.ParseDuration(d.Val())
		if err != nil {
			return nil, d.WrapErr(err)
		}
		module.Validity = caddy.Duration(dur)
	}

	return module, nil
}

func parseLatency(critic *latency.Critic, d *caddyfile.Dispenser, _ *[]caddyconfig.Warning) (caddy.Module, error) {
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

			validity, err := caddy.ParseDuration(d.Val())
			if err != nil {
				return nil, d.WrapErr(err)
			}

			critic.Validity = caddy.Duration(validity)
		case "query_threshold":
			if !d.NextArg() {
				return nil, d.ArgErr()
			}

			threshold, err := caddy.ParseDuration(d.Val())
			if err != nil {
				return nil, d.WrapErr(err)
			}

			critic.QueryThreshold = caddy.Duration(threshold)
		default:
			return nil, d.ArgErr()
		}

		if !d.NextLine() {
			return nil, d.EOFErr()
		}
	}

	return critic, nil
}
