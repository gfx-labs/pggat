package main

import (
	"context"

	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat/gatling"
	"git.tuxpa.in/a/zlog/log"
)

// test config, should be changed
const CONFIG = "./config_data.yml"

func main() {
	conf, err := config.Load(CONFIG)
	if err != nil {
		panic(err)
	}
	g := gatling.NewGatling(conf)
	log.Println("listening on port", conf.General.Port)
	err = g.ListenAndServe(context.Background())
	if err != nil {
		panic(err)
	}
}
