package main

import (
	"net/http"
	_ "net/http/pprof"
	"os"

	"tuxpa.in/a/zlog/log"

	"pggat2/lib/gat/configs/pgbouncer"
	"pggat2/lib/gat/configs/zalando"
)

func main() {
	go func() {
		panic(http.ListenAndServe(":8080", nil))
	}()

	log.Printf("Starting pggat...")

	if len(os.Args) == 2 {
		log.Printf("running in pgbouncer compatibility mode")
		conf, err := pgbouncer.Load(os.Args[1])
		if err != nil {
			panic(err)
		}

		err = conf.ListenAndServe()
		if err != nil {
			panic(err)
		}
		return
	}

	log.Printf("running in zalando compatibility mode")

	conf, err := zalando.Load()
	if err != nil {
		panic(err)
	}

	err = conf.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
