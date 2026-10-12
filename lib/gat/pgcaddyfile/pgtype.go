package pgcaddyfile

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/configbuilder"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile/blocktypes"
	_ "github.com/caddyserver/caddy/v2/modules/standard"

	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/matchers"
	"gfx.cafe/gfx/pggat/lib/gat/ssl/servers/self_signed"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func init() {
	blocktypes.RegisterBlockType("pg", setup)
}

func setup(builder *configbuilder.Builder, blocks []caddyfile.ServerBlock, _ map[string]any) ([]caddyconfig.Warning, error) {
	var warnings []caddyconfig.Warning
	app := gat.App{
		Config: gat.Config{
			StatLogPeriod: caddy.Duration(time.Minute),
		},
	}

	for _, block := range blocks {
		if len(block.Keys) == 0 {
			for _, segment := range block.Segments {
				d := caddyfile.NewDispenser(segment)
				if !d.Next() {
					continue
				}
				if d.Val() != "stat_log_period" {
					return warnings, d.SyntaxErr("[pg] options")
				}
				if !d.NextArg() {
					return warnings, d.ArgErr()
				}
				period, err := time.ParseDuration(d.Val())
				if err != nil {
					return warnings, d.WrapErr(err)
				}
				if period <= 0 {
					return warnings, d.Err("stat_log_period must be greater than zero")
				}
				if d.CountRemainingArgs() > 0 {
					return warnings, d.ArgErr()
				}
				if d.NextBlock(0) {
					return warnings, d.Err("stat_log_period does not accept a block")
				}
				app.StatLogPeriod = caddy.Duration(period)
			}

			continue
		}

		if err := parseServerBlock(&app, block, &warnings); err != nil {
			return warnings, err
		}
	}

	if err := builder.CreateApp("pggat", &app); err != nil {
		return warnings, err
	}
	return warnings, nil
}

func parseServerBlock(app *gat.App, block caddyfile.ServerBlock, warnings *[]caddyconfig.Warning) error {
	var server gat.ServerConfig
	server.Listen = make([]gat.ListenerConfig, 0, len(block.Keys))
	for _, key := range block.Keys {
		listen := gat.ListenerConfig{
			Address:        key.Text,
			MaxConnections: 1000,
		}
		server.Listen = append(server.Listen, listen)
	}

	var namedMatchers map[string]json.RawMessage

	for _, segment := range block.Segments {
		d := caddyfile.NewDispenser(segment)
		if !d.Next() {
			continue
		}
		directive := d.Val()
		switch {
		case directive == directiveSSL:
			var val json.RawMessage
			if !d.NextArg() {
				// self signed ssl
				val = caddyconfig.JSONModuleObject(
					self_signed.Server{},
					"provider",
					"self_signed",
					warnings,
				)
			} else {
				var err error
				val, err = UnmarshalDirectiveJSONModuleObject(
					d,
					SSLServer,
					"provider",
					warnings,
				)
				if err != nil {
					return err
				}
			}

			// set
			for i := range server.Listen {
				if server.Listen[i].SSL != nil {
					return d.Err("duplicate ssl directive")
				}
				server.Listen[i].SSL = val
			}

			if d.CountRemainingArgs() > 0 {
				return d.ArgErr()
			}
		case directive == "max_connections":
			if !d.NextArg() {
				return d.ArgErr()
			}

			maxConnections, err := strconv.Atoi(d.Val())
			if err != nil {
				return d.Err(err.Error())
			}

			for i := range server.Listen {
				server.Listen[i].MaxConnections = maxConnections
			}

			if d.CountRemainingArgs() > 0 {
				return d.ArgErr()
			}
		case strings.HasPrefix(directive, "@"):
			name := strings.TrimPrefix(directive, "@")
			if _, ok := namedMatchers[name]; ok {
				return d.Errf(`duplicate named matcher "%s"`, name)
			}

			var matcher json.RawMessage

			// read named matcher
			if d.NextArg() {
				// inline
				var err error
				matcher, err = UnmarshalDirectiveJSONModuleObject(
					d,
					Matcher,
					"matcher",
					warnings,
				)
				if err != nil {
					return err
				}
			} else {
				// block
				if !d.NextBlock(0) {
					return d.ArgErr()
				}

				var and matchers.And

				for d.Val() != "}" {

					cond, err := UnmarshalDirectiveJSONModuleObject(
						d,
						Matcher,
						"matcher",
						warnings,
					)
					if err != nil {
						return err
					}
					and.And = append(and.And, cond)

					if !d.NextLine() {
						return d.EOFErr()
					}
				}

				switch len(and.And) {
				case 0:
					matcher = nil
				case 1:
					matcher = and.And[0]
				default:
					matcher = caddyconfig.JSONModuleObject(
						and,
						"matcher",
						"and",
						warnings,
					)
				}
			}

			if d.CountRemainingArgs() > 0 {
				return d.ArgErr()
			}

			if namedMatchers == nil {
				namedMatchers = make(map[string]json.RawMessage)
			}
			namedMatchers[name] = matcher
		default:
			unmarshaller, ok := LookupDirective(Handler, d.Val())
			if !ok {
				return d.Errf(`unknown handler "%s"`, d.Val())
			}

			var route gat.RouteConfig

			// try to read matcher
			if d.NextArg() {
				matcher := d.Val()
				switch {
				case strings.HasPrefix(matcher, "@"): // named matcher
					route.Match, ok = namedMatchers[strings.TrimPrefix(matcher, "@")]
					if !ok {
						return d.Errf(`unknown named matcher "%s"`, matcher)
					}
				case strings.HasPrefix(matcher, "/"): // database
					route.Match = caddyconfig.JSONModuleObject(
						matchers.Database{
							Database: strutil.Matcher(strings.TrimPrefix(matcher, "/")),
						},
						"matcher",
						"database",
						warnings,
					)
				case matcher == "*":
					route.Match = nil // wildcard
				default:
					d.Prev()
				}
			}

			var err error
			route.Handle, err = unmarshaller.JSONModuleObject(
				d,
				Handler,
				"handler",
				warnings,
			)
			if err != nil {
				return err
			}

			if d.CountRemainingArgs() > 0 {
				return d.ArgErr()
			}

			server.Routes = append(server.Routes, route)
		}
	}

	app.Servers = append(app.Servers, server)
	return nil
}
