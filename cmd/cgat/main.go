package main

import (
	"net"

	"pggat2/lib/backend/backends/v0"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:5432")
	if err != nil {
		panic(err)
	}
	srv, err := backends.NewServer(conn)
	if err != nil {
		panic(err)
	}
	_ = srv
}
