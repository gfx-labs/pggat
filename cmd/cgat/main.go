package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"

	"pggat2/lib/gat"
	"pggat2/lib/gat/pools/transaction"
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
	pool := transaction.NewPool()
	postgres.AddPool("uniswap", pool)
	pool.AddRecipe("localhost", gat.Recipe{
		Database:       "uniswap",
		Address:        "localhost:5432",
		User:           "postgres",
		Password:       "password",
		MinConnections: 5,
		MaxConnections: 5,
	})

	log.Println("Listening on :6432")

	err := pooler.ListenAndServe(":6432")
	if err != nil {
		panic(err)
	}
}
