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
	pooler.Mount("uniswap", transaction.NewPool())

	log.Println("Listening on :6432")

	err := pooler.ListenAndServe(":6432")
	if err != nil {
		panic(err)
	}
}
