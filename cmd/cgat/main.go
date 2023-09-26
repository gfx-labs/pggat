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

func addSSLEndpoint(server *gat.Server) error {
	// back up ssl endpoint (for modules that don't have endpoints by default such as discovery)
	cert, err := certs.SelfSign()
	if err != nil {
		return err
	}
	server.AddModule(&net_listener.Module{
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
	})

	return nil
}

func addEnvModule(server *gat.Server, mode string) error {
	switch mode {
	case "pggat":
		conf, err := pgbouncer.Load(os.Args[1])
		if err != nil {
			return err
		}

		server.AddModule(&pgbouncer.Module{
			Config: conf,
		})
	case "pgbouncer":
		conf, err := pgbouncer.Load(os.Args[1])
		if err != nil {
			return err
		}

		server.AddModule(&pgbouncer.Module{
			Config: conf,
		})
	case "pgbouncer_spilo":
		conf, err := zalando.Load()
		if err != nil {
			return err
		}

		server.AddModule(&zalando.Module{
			Config: conf,
		})
	case "zalando_kubernetes_operator":
		conf, err := zalando_operator_discovery.Load()
		if err != nil {
			return err
		}

		server.AddModule(&zalando_operator_discovery.Module{
			Config: conf,
		})
		return addSSLEndpoint(server)
	case "google_cloud_sql":
		conf, err := cloud_sql_discovery.Load()
		if err != nil {
			return err
		}

		server.AddModule(&cloud_sql_discovery.Module{
			Config: conf,
		})
		return addSSLEndpoint(server)
	case "digitalocean_databases":
		conf, err := digitalocean_discovery.Load()
		if err != nil {
			return err
		}

		server.AddModule(&digitalocean_discovery.Module{
			Config: conf,
		})
		return addSSLEndpoint(server)
	default:
		return errors.New("Unknown PGGAT_RUN_MODE: " + mode)
	}

	return nil
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

	// load and add main module
	if err := addEnvModule(&server, runMode); err != nil {
		panic(err)
	}

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
