package gatcaddyfile

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/matchers"
	"gfx.cafe/gfx/pggat/lib/gat/ssl/servers/self_signed"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func init() {
	caddyconfig.RegisterAdapter("gatfile", caddyfile.Adapter{ServerType: ServerType{}})
}

type ServerType struct{}

func (ServerType) Setup(blocks []caddyfile.ServerBlock, m map[string]any) (*caddy.Config, []caddyconfig.Warning, error) {
	var config caddy.Config
	var warnings []caddyconfig.Warning

	app := gat.App{
		Config: gat.Config{
			StatLogPeriod: caddy.Duration(1 * time.Minute),
		},
	}

	for i, block := range blocks {
		if i == 0 && len(block.Keys) == 0 {
			// global options
			for _, segment := range block.Segments {
				d := caddyfile.NewDispenser(segment)
				if !d.Next() {
					continue
				}
				directive := d.Val()
				switch {
				case directive == "stat_log_period":
					if !d.NextArg() {
						return nil, nil, d.ArgErr()
					}

					period, err := time.ParseDuration(d.Val())
					if err != nil {
						return nil, nil, d.WrapErr(err)
					}

					app.StatLogPeriod = caddy.Duration(period)
				default:
					return nil, nil, d.SyntaxErr("global options")
				}

				if d.CountRemainingArgs() > 0 {
					return nil, nil, d.ArgErr()
				}
			}

			continue
		}

		var server gat.ServerConfig

		server.Listen = make([]gat.ListenerConfig, 0, len(block.Keys))
		for _, key := range block.Keys {
			listen := gat.ListenerConfig{
				Address: key,
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
			case directive == "ssl":
				var val json.RawMessage
				if !d.NextArg() {
					// self signed ssl
					val = caddyconfig.JSONModuleObject(
						self_signed.Server{},
						"provider",
						"self_signed",
						&warnings,
					)
				} else {
					var err error
					val, err = UnmarshalDirectiveJSONModuleObject(
						d,
						SSLServer,
						"provider",
						&warnings,
					)
					if err != nil {
						return nil, nil, err
					}
				}

				// set
				for i := range server.Listen {
					if server.Listen[i].SSL != nil {
						return nil, nil, d.Err("duplicate ssl directive")
					}
					server.Listen[i].SSL = val
				}

				if d.CountRemainingArgs() > 0 {
					return nil, nil, d.ArgErr()
				}
			case strings.HasPrefix(directive, "@"):
				name := strings.TrimPrefix(directive, "@")
				if _, ok := namedMatchers[name]; ok {
					return nil, nil, d.Errf(`duplicate named matcher "%s"`, name)
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
						&warnings,
					)
					if err != nil {
						return nil, nil, err
					}
				} else {
					// block
					if !d.NextBlock(0) {
						return nil, nil, d.ArgErr()
					}

					var and matchers.And

					for {
						if d.Val() == "}" {
							break
						}

						cond, err := UnmarshalDirectiveJSONModuleObject(
							d,
							Matcher,
							"matcher",
							&warnings,
						)
						if err != nil {
							return nil, nil, err
						}
						and.And = append(and.And, cond)

						if !d.NextLine() {
							return nil, nil, d.EOFErr()
						}
					}

					if len(and.And) == 0 {
						matcher = nil
					} else if len(and.And) == 1 {
						matcher = and.And[0]
					} else {
						matcher = caddyconfig.JSONModuleObject(
							and,
							Matcher,
							"matcher",
							&warnings,
						)
					}
				}

				if d.CountRemainingArgs() > 0 {
					return nil, nil, d.ArgErr()
				}

				if namedMatchers == nil {
					namedMatchers = make(map[string]json.RawMessage)
				}
				namedMatchers[name] = matcher
			default:
				unmarshaller, ok := LookupDirective(Handler, d.Val())
				if !ok {
					return nil, nil, d.Errf(`unknown handler "%s"`, d.Val())
				}

				var route gat.RouteConfig

				// try to read matcher
				if d.NextArg() {
					matcher := d.Val()
					switch {
					case strings.HasPrefix(matcher, "@"): // named matcher
						route.Match, ok = namedMatchers[strings.TrimPrefix(matcher, "@")]
						if !ok {
							return nil, nil, d.Errf(`unknown named matcher "%s"`, matcher)
						}
					case strings.HasPrefix(matcher, "/"): // database
						route.Match = caddyconfig.JSONModuleObject(
							matchers.Database{
								Database: strutil.Matcher(strings.TrimPrefix(matcher, "/")),
							},
							"matcher",
							"database",
							&warnings,
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
					&warnings,
				)
				if err != nil {
					return nil, nil, err
				}

				if d.CountRemainingArgs() > 0 {
					return nil, nil, d.ArgErr()
				}

				server.Routes = append(server.Routes, route)
			}
		}

		app.Servers = append(app.Servers, server)
	}

	if config.AppsRaw == nil {
		config.AppsRaw = make(caddy.ModuleMap)
	}
	config.AppsRaw[string(app.CaddyModule().ID)] = caddyconfig.JSON(app, &warnings)

	return &config, warnings, nil
}
