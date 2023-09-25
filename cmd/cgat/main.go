package main

import (
	"errors"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"tuxpa.in/a/zlog/log"

	"pggat/lib/gat"
	"pggat/lib/gat/metrics"
	"pggat/lib/gat/modules/cloud_sql_discovery"
	"pggat/lib/gat/modules/digitalocean_discovery"
	"pggat/lib/gat/modules/pgbouncer"
	"pggat/lib/gat/modules/ssl_endpoint"
	"pggat/lib/gat/modules/zalando"
	"pggat/lib/gat/modules/zalando_operator_discovery"
)

func loadModule(mode string) (gat.Module, error) {
	switch mode {
	case "pggat":
		conf, err := pgbouncer.Load(os.Args[1])
		if err != nil {
			return nil, err
		}
		return pgbouncer.NewModule(conf)
	case "pgbouncer":
		conf, err := pgbouncer.Load(os.Args[1])
		if err != nil {
			return nil, err
		}
		return pgbouncer.NewModule(conf)
	case "pgbouncer_spilo":
		conf, err := zalando.Load()
		if err != nil {
			return nil, err
		}
		return zalando.NewModule(conf)
	case "zalando_kubernetes_operator":
		conf, err := zalando_operator_discovery.Load()
		if err != nil {
			return nil, err
		}
		return zalando_operator_discovery.NewModule(conf)
	case "google_cloud_sql":
		conf, err := cloud_sql_discovery.Load()
		if err != nil {
			return nil, err
		}
		return cloud_sql_discovery.NewModule(conf)
	case "digitalocean_databases":
		conf, err := digitalocean_discovery.Load()
		if err != nil {
			return nil, err
		}
		return digitalocean_discovery.NewModule(conf)
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

	// load and add main module
	module, err := loadModule(runMode)
	if err != nil {
		panic(err)
	}
	server.AddModule(module)

	// back up ssl endpoint (for modules that don't have endpoints by default such as discovery)
	ep, err := ssl_endpoint.NewModule()
	if err != nil {
		panic(err)
	}
	server.AddModule(ep)

	go func() {
		var m metrics.Server
		for {
			time.Sleep(1 * time.Minute)
			server.ReadMetrics(&m)
			log.Printf("%s", m.String())
			m.Clear()
		}
	}()

	err = server.ListenAndServe()
	if err != nil {
		panic(err)
	}
	return
}
