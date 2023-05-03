package main

import (
	"net"

	"pggat2/lib/backend/backends/v0"
	"pggat2/lib/pnet/packet"
	"pggat2/lib/request"
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
	conn, err := net.Dial("tcp", "localhost:5432")
	if err != nil {
		panic(err)
	}
	server, err := backends.NewServer(conn)
	if err != nil {
		panic(err)
	}
	var builder packet.Builder
	builder.Type(packet.Query)
	builder.String("select 1")
	server.Request(request.NewSimple(builder.Raw()))
}
