package main

import (
	"crypto/tls"
	"errors"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tuxpa.in/a/zlog/log"

	"gfx.cafe/gfx/pggat/lib/bouncer/frontends/v0"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/gat/modules/cloud_sql_discovery"
	"gfx.cafe/gfx/pggat/lib/gat/modules/digitalocean_discovery"
	"gfx.cafe/gfx/pggat/lib/gat/modules/net_listener"
	"gfx.cafe/gfx/pggat/lib/gat/modules/pgbouncer"
	"gfx.cafe/gfx/pggat/lib/gat/modules/zalando"
	"gfx.cafe/gfx/pggat/lib/gat/modules/zalando_operator_discovery"
	"gfx.cafe/gfx/pggat/lib/util/certs"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func createSSLModule() (gat.Module, error) {
	// back up ssl endpoint (for modules that don't have endpoints by default such as discovery)
	cert, err := certs.SelfSign()
	if err != nil {
		return nil, err
	}

	return &net_listener.Module{
		Config: net_listener.Config{
			Network: "tcp",
			Address: ":5432",
			AcceptOptions: frontends.AcceptOptions{
				SSLRequired: false,
				SSLConfig: &tls.Config{
					Certificates: []tls.Certificate{cert},
				},
				AllowedStartupOptions: []strutil.CIString{
					strutil.MakeCIString("client_encoding"),
					strutil.MakeCIString("datestyle"),
					strutil.MakeCIString("timezone"),
					strutil.MakeCIString("standard_conforming_strings"),
					strutil.MakeCIString("application_name"),
					strutil.MakeCIString("extra_float_digits"),
					strutil.MakeCIString("options"),
				},
			},
		},
	}, nil
}

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

	// load modules
	var modules []gat.Module

	module, err := loadModule(runMode)
	if err != nil {
		panic(err)
	}
	modules = append(modules, module)

	if _, ok := module.(gat.Listener); !ok {
		endpoint, err := createSSLModule()
		if err != nil {
			panic(err)
		}
		modules = append(modules, endpoint)
	}

	server := gat.NewServer(modules...)

	// handle interrupts
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c

		if err := server.Stop(); err != nil {
			log.Printf("error stopping: %v", err)
		}

		os.Exit(0)
	}()

	go func() {
		var m metrics.Server
		for {
			time.Sleep(1 * time.Minute)
			server.ReadMetrics(&m)
			log.Printf("%s", m.String())
			m.Clear()
		}
	}()

	if err := server.Start(); err != nil {
		panic(err)
	}

	select {}
}
