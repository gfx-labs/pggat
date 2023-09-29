package gatcaddyfile

import (
	"fmt"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	"gfx.cafe/gfx/pggat/lib/gat"
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

		server.Match = MatcherFromConnectionStrings(block.Keys, &warnings)

		for _, segment := range block.Segments {
			d := caddyfile.NewDispenser(segment)
			if !d.Next() {
				continue
			}
			switch d.Val() {
			case "pgbouncer":
				var route gat.RouteConfig
				if d.CountRemainingArgs() == 2 {
					d.Next()
					route.Match = MatcherFromConnectionString(d.Val(), &warnings)
				}
				// TODO(garet) read actual value
				server.Routes = append(server.Routes, route)
			case "discovery":
				// TODO(garet)
			default:
				return nil, nil, fmt.Errorf("unknown directive: %s", d.Val())
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
