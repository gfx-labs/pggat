package main

import "pggat2/lib/frontend/frontends/v0"

func main() {
	frontend, err := frontends.NewFrontend()
	if err != nil {
		panic(err)
	}
	err = frontend.Run()
	if err != nil {
		panic(err)
	}
	/*
		conn, err := net.Dial("tcp", "localhost:5432")
		if err != nil {
			panic(err)
		}
		server, err := backends.NewServer(conn)
		if err != nil {
			panic(err)
		}
		_ = server
		_ = conn.Close()*/
}
