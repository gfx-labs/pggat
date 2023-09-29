package main

import (
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	caddycmd "github.com/caddyserver/caddy/v2/cmd"

	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pgbouncer_spilo"
	"gfx.cafe/gfx/pggat/lib/util/dur"

	"gfx.cafe/util/go/gun"
)

func init() {
	caddycmd.RegisterCommand(caddycmd.Command{
		Name:  "pgbouncer-spilo",
		Short: "Runs in pgbouncer-spilo compatibility mode",
		Func:  runPgbouncerSpilo,
	})
}

func runPgbouncerSpilo(flags caddycmd.Flags) (int, error) {
	caddy.TrapSignals()

	var config pgbouncer_spilo.Config
	gun.Load(&config)

	var pggat gat.Config
	pggat.StatLogPeriod = dur.Duration(1 * time.Minute)

	var server gat.ServerConfig
	server.Listen = config.Listen()
	server.Routes = append(server.Routes, gat.RouteConfig{
		Handle: caddyconfig.JSONModuleObject(
			pgbouncer_spilo.Module{
				Config: config,
			},
			"handler",
			"pgbouncer_spilo",
			nil,
		),
	})
	pggat.Servers = append(pggat.Servers, server)

	caddyConfig := caddy.Config{
		AppsRaw: caddy.ModuleMap{
			"pggat": caddyconfig.JSON(pggat, nil),
		},
	}

	if err := caddy.Run(&caddyConfig); err != nil {
		return caddy.ExitCodeFailedStartup, err
	}

	select {}
}
