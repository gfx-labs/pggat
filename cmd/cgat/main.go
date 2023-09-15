package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"

	"tuxpa.in/a/zlog/log"

	"pggat/lib/gat/modes/digitalocean_discovery"
	"pggat/lib/gat/modes/pgbouncer"
	"pggat/lib/gat/modes/zalando"
	"pggat/lib/gat/modes/zalando_operator_discovery"
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

	if os.Getenv("CONNECTION_POOLER_MODE") != "" {
		log.Printf("running in zalando compatibility mode")

		conf, err := zalando.Load()
		if err != nil {
			panic(err)
		}

		err = conf.ListenAndServe()
		if err != nil {
			panic(err)
		}
		return
	}

	if os.Getenv("PGGAT_DO_API_KEY") != "" {
		log.Printf("running in digitalocean discovery mode")

		conf, err := digitalocean_discovery.Load()
		if err != nil {
			panic(err)
		}

		err = conf.ListenAndServe()
		if err != nil {
			panic(err)
		}
		return
	}

	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" && os.Getenv("KUBERNETES_SERVICE_PORT") != "" {
		log.Printf("running in zalando operator discovery mode")
		conf, err := zalando_operator_discovery.Load()
		if err != nil {
			panic(err)
		}

		err = conf.ListenAndServe()
		if err != nil {
			panic(err)
		}
		return
	}

	panic(fmt.Sprintf("usage: %s <config>", os.Args[0]))
}
