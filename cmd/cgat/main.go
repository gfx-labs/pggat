package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

	"pggat2/lib/gat"
	"pggat2/lib/gat/pools/transaction"
	"pggat2/lib/rob"
)

func main() {
	go func() {
		panic(http.ListenAndServe(":8080", nil))
	}()

	log.Println("Starting pggat...")

	pooler := gat.NewPooler()

	// create user
	postgres := gat.NewUser("pw")
	pooler.AddUser("postgres", postgres)

	// create pool
	rawPool := transaction.NewPool()
	pool := gat.NewPool(rawPool)
	postgres.AddPool("uniswap", pool)
	pool.AddRecipe("localhost", gat.TCPRecipe{
		Database:       "uniswap",
		Address:        "localhost:5432",
		User:           "postgres",
		Password:       "password",
		MinConnections: 0,
		MaxConnections: 5,
	})
	pool.Scale(1)

	go func() {
		var metrics rob.Metrics

		for {
			time.Sleep(1 * time.Second)
			rawPool.ReadSchedulerMetrics(&metrics)
			log.Println(metrics.String())
		}
	}()

	log.Println("Listening on :6432")

	err := pooler.ListenAndServe(":6432")
	if err != nil {
		panic(err)
	}
}
