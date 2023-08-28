package main

import (
	"crypto/tls"
	"net/http"
	_ "net/http/pprof"

	"tuxpa.in/a/zlog/log"

	"pggat2/lib/auth/credentials"
	"pggat2/lib/bouncer"
	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/bouncer/frontends/v0"
	"pggat2/lib/gat"
	"pggat2/lib/gat/pools/session"
)

func main() {
	go func() {
		panic(http.ListenAndServe(":8080", nil))
	}()

	log.Printf("Starting pggat...")

	g := new(gat.Gat)
	g.TestPool = session.NewPool(gat.PoolOptions{
		Credentials: credentials.Cleartext{
			Username: "postgres",
			Password: "password",
		},
	})
	g.TestPool.AddRecipe("test", gat.Recipe{
		Dialer: gat.NetDialer{
			Network: "tcp",
			Address: "localhost:5432",

			AcceptOptions: backends.AcceptOptions{
				SSLMode: bouncer.SSLModeAllow,
				SSLConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
				Credentials: credentials.Cleartext{
					Username: "postgres",
					Password: "password",
				},
				Database: "postgres",
			},
		},
		MinConnections: 1,
		MaxConnections: 1,
	})
	err := gat.ListenAndServe("tcp", ":6432", frontends.AcceptOptions{}, g)
	if err != nil {
		panic(err)
	}

	/*
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
	*/
}
