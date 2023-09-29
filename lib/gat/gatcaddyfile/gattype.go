package gatcaddyfile

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/matchers"
	"gfx.cafe/gfx/pggat/lib/gat/ssl/servers/self_signed"
)

func init() {
	caddyconfig.RegisterAdapter("caddyfile", caddyfile.Adapter{ServerType: ServerType{}})
}

type ServerType struct{}

func (ServerType) Setup(blocks []caddyfile.ServerBlock, m map[string]any) (*caddy.Config, []caddyconfig.Warning, error) {
	var config caddy.Config
	var warnings []caddyconfig.Warning

	var app gat.App

	for _, block := range blocks {
		var server gat.ServerConfig

		server.Listen = make([]gat.ListenerConfig, 0, len(block.Keys))
		for _, key := range block.Keys {
			var listen gat.ListenerConfig
			if strings.HasPrefix(key, "/") {
				listen = gat.ListenerConfig{
					Network: "unix",
					Address: key,
				}
			} else {
				listen = gat.ListenerConfig{
					Network: "tcp",
					Address: key,
				}
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
					unmarshaller, ok := sslServers[d.Val()]
					if !ok {
						return nil, nil, d.Errf(`unknown ssl server "%s"`, d.Val())
					}

					var err error
					val, err = unmarshaller.JSONModuleObject(
						d,
						"pggat.ssl.servers",
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
				} else {
					// block
					if !d.NextBlock(0) {
						return nil, nil, d.ArgErr()
					}

					for {
						if d.Val() == "}" {
							break
						}

						log.Println(d.Val())
						for d.NextArg() {
							log.Println(d.Val())
						}

						if !d.NextLine() {
							return nil, nil, d.EOFErr()
						}
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
				unmarshaller, ok := handlers[d.Val()]
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
								Database: strings.TrimPrefix(matcher, "/"),
							},
							"matcher",
							"database",
							&warnings,
						)
					default:
						d.Prev()
					}
				}

				var err error
				route.Handle, err = unmarshaller.JSONModuleObject(
					d,
					"pggat.handlers",
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
