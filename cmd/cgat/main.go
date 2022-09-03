package main

import (
	"context"
	"log"

	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
)

// test config, should be changed
const CONFIG = "./config_data.toml"

func main() {
	conf, err := config.Load(CONFIG)
	if err != nil {
		panic(err)
	}
	gatling := gat.NewGatling()
	err = gatling.ApplyConfig(conf)
	if err != nil {
		panic(err)
	}

	log.Println("listening on port", conf.General.Port)
	err = gatling.ListenAndServe(context.Background())
	if err != nil {
		panic(err)
	}
}
