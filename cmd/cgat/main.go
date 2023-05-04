package main

import (
	"net"

	"pggat2/lib/backend/backends/v0"
)

func main() {
	/*
		frontend, err := frontends.NewFrontend()
		if err != nil {
			panic(err)
		}
		err = frontend.Run()
		if err != nil {
			panic(err)
		}
	*/
	for i := 0; i < 1000; i++ {
		conn, err := net.Dial("tcp", "localhost:5432")
		if err != nil {
			panic(err)
		}
		server, err := backends.NewServer(conn)
		if err != nil {
			panic(err)
		}
		_ = server
		_ = conn.Close()
	}
}
