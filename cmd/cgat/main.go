package main

import (
	"net/http"
	_ "net/http/pprof"
	"os"

	"tuxpa.in/a/zlog/log"

	"pggat/lib/gat/modes/cloud_sql_discovery"
	"pggat/lib/gat/modes/digitalocean_discovery"
	"pggat/lib/gat/modes/pgbouncer"
	"pggat/lib/gat/modes/zalando"
	"pggat/lib/gat/modes/zalando_operator_discovery"
)

func main() {
	go func() {
		panic(http.ListenAndServe(":8080", nil))
	}()

	runMode := os.Getenv("PGGAT_RUN_MODE")
	if runMode == "" {
		runMode = "pgbouncer"
	}

	log.Printf("Starting pggat (%s)...", runMode)

	// TODO: this really should load a dynamically registered module
	var conf interface {
		ListenAndServe() error
	}
	var err error
	switch runMode {
	case "pggat":
		conf, err = pgbouncer.Load(os.Args[1])
	case "pgbouncer":
		conf, err = pgbouncer.Load(os.Args[1])
	case "pgbouncer_spilo":
		conf, err = zalando.Load()
	case "zalando_kubernetes_operator":
		conf, err = zalando_operator_discovery.Load()
	case "google_cloud_sql":
		conf, err = cloud_sql_discovery.Load()
	case "digitalocean_databases":
		conf, err = digitalocean_discovery.Load()
	default:
		panic("Unknown PGGAT_RUN_MODE: " + runMode)
	}
	if err != nil {
		panic(err)
	}

	err = conf.ListenAndServe()
	if err != nil {
		panic(err)
	}
	return
}
