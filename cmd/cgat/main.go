package main

import (
	"errors"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tuxpa.in/a/zlog/log"

	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/gat/modules/cloud_sql_discovery"
	"gfx.cafe/gfx/pggat/lib/gat/modules/digitalocean_discovery"
	"gfx.cafe/gfx/pggat/lib/gat/modules/pgbouncer"
	"gfx.cafe/gfx/pggat/lib/gat/modules/ssl_endpoint"
	"gfx.cafe/gfx/pggat/lib/gat/modules/zalando"
	"gfx.cafe/gfx/pggat/lib/gat/modules/zalando_operator_discovery"
)

func loadModule(mode string) (gat.Module, error) {
	switch mode {
	case "pggat":
		conf, err := pgbouncer.Load(os.Args[1])
		if err != nil {
			return nil, err
		}
		return &pgbouncer.Module{
			Config: conf,
		}, nil
	case "pgbouncer":
		conf, err := pgbouncer.Load(os.Args[1])
		if err != nil {
			return nil, err
		}
		return &pgbouncer.Module{
			Config: conf,
		}, nil
	case "pgbouncer_spilo":
		conf, err := zalando.Load()
		if err != nil {
			return nil, err
		}
		return &zalando.Module{
			Config: conf,
		}, nil
	case "zalando_kubernetes_operator":
		conf, err := zalando_operator_discovery.Load()
		if err != nil {
			return nil, err
		}
		return &zalando_operator_discovery.Module{
			Config: conf,
		}, nil
	case "google_cloud_sql":
		conf, err := cloud_sql_discovery.Load()
		if err != nil {
			return nil, err
		}
		return &cloud_sql_discovery.Module{
			Config: conf,
		}, nil
	case "digitalocean_databases":
		conf, err := digitalocean_discovery.Load()
		if err != nil {
			return nil, err
		}
		return &digitalocean_discovery.Module{
			Config: conf,
		}, nil
	default:
		return nil, errors.New("Unknown PGGAT_RUN_MODE: " + mode)
	}
}

func main() {
	go func() {
		panic(http.ListenAndServe(":8080", nil))
	}()

	runMode := os.Getenv("PGGAT_RUN_MODE")
	if runMode == "" {
		runMode = "pgbouncer"
	}

	log.Printf("Starting pggat (%s)...", runMode)

	var server gat.Server
	defer func() {
		if err := server.Stop(); err != nil {
			log.Printf("error stopping: %v", err)
		}
	}()

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c

		if err := server.Stop(); err != nil {
			log.Printf("error stopping: %v", err)
		}
	}()

	// load and add main module
	module, err := loadModule(runMode)
	if err != nil {
		panic(err)
	}
	server.AddModule(module)

	// back up ssl endpoint (for modules that don't have endpoints by default such as discovery)
	server.AddModule(&ssl_endpoint.Module{})

	go func() {
		var m metrics.Server
		for {
			time.Sleep(1 * time.Minute)
			server.ReadMetrics(&m)
			log.Printf("%s", m.String())
			m.Clear()
		}
	}()

	err = server.Start()
	if err != nil {
		panic(err)
	}
	return
}
