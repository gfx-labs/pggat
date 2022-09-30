package main

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat/gatling"
	"gfx.cafe/util/go/graceful"
	"git.tuxpa.in/a/zlog/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe("localhost:6060", nil)

	confLocation := os.Getenv("CONFIG_LOCATION")
	if confLocation == "" {
		confLocation = "./config_data.yml"
	}
	conf, err := config.Load(confLocation)
	if err != nil {
		panic(err)
	}
	g := gatling.NewGatling(conf)
	log.Println("listening on port", conf.General.Port)
	graceful.Handler(g.ListenAndServe, func(ctx context.Context) error {
		log.Println("shutting down in 3 seconds...")
		time.Sleep(3 * time.Second)
		return nil
	})
}
