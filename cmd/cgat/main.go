package main

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"time"

	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat/gatling"
	"gfx.cafe/util/go/graceful"
	"git.tuxpa.in/a/zlog/log"
)

// test config, should be changed
const CONFIG = "./config_data.yml"

func main() {
	//zlog.SetGlobalLevel(zlog.PanicLevel)
	go func() {

		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	conf, err := config.Load(CONFIG)
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
