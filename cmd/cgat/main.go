package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"

	"pggat2/lib/gat"
	"pggat2/lib/gat/configs/pgbouncer"
)

func main() {
	go func() {
		panic(http.ListenAndServe(":8080", nil))
	}()

	log.Println("Starting pggat...")

	conf, err := pgbouncer.Load()
	if err != nil {
		panic(err)
	}

	pooler := gat.NewPooler()

	err = conf.ListenAndServe(pooler)
	if err != nil {
		panic(err)
	}
}
