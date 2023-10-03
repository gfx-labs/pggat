package main

import (
	"errors"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	caddycmd "github.com/caddyserver/caddy/v2/cmd"

	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pgbouncer"
	"gfx.cafe/gfx/pggat/lib/util/dur"
)

func init() {
	caddycmd.RegisterCommand(caddycmd.Command{
		Name:  "pgbouncer",
		Usage: "<config file>",
		Short: "Runs in pgbouncer compatibility mode",
		Func:  runPgbouncer,
	})
}

func runPgbouncer(flags caddycmd.Flags) (int, error) {
	caddy.TrapSignals()

	file := flags.Arg(0)
	if file == "" {
		return caddy.ExitCodeFailedStartup, errors.New("usage: pgbouncer <config file>")
	}

	config, err := pgbouncer.Load(file)
	if err != nil {
		return caddy.ExitCodeFailedStartup, err
	}

	var pggat gat.Config
	pggat.StatLogPeriod = dur.Duration(time.Second)

	var server gat.ServerConfig
	server.Listen = config.Listen()
	server.Routes = append(server.Routes, gat.RouteConfig{
		Handle: caddyconfig.JSONModuleObject(
			pgbouncer.Module{
				ConfigFile: file,
			},
			"handler",
			"pgbouncer",
			nil,
		),
	})
	pggat.Servers = append(pggat.Servers, server)

	caddyConfig := caddy.Config{
		AppsRaw: caddy.ModuleMap{
			"pggat": caddyconfig.JSON(pggat, nil),
		},
	}

	if err = caddy.Run(&caddyConfig); err != nil {
		return caddy.ExitCodeFailedStartup, err
	}

	select {}
}
