package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

	"pggat2/lib/gat/configs/pgbouncer"
)

func main() {
	go func() {
		panic(http.ListenAndServe(":8080", nil))
	}()

	log.Println("Starting pggat...")

	if len(os.Args) < 2 {
		panic("usage: pggat <config>")
	}

	conf, err := pgbouncer.Load(os.Args[1])
	if err != nil {
		panic(err)
	}

	err = conf.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
