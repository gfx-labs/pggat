package main

import (
	"net"
	"net/http"
	_ "net/http/pprof"

	"pggat2/lib/backend/backends/v0"
	"pggat2/lib/frontend/frontends/v0"
	"pggat2/lib/pnet"
	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v2"
)

type job struct {
	rw   pnet.ReadWriter
	done chan<- struct{}
}

func testServer(r rob.Scheduler) {
	conn, err := net.Dial("tcp", "localhost:5432")
	if err != nil {
		panic(err)
	}
	server := backends.NewServer(conn)
	if server == nil {
		panic("failed to connect to server")
	}

	sink := r.NewSink(0)
	for {
		j := sink.Read().(job)
		server.Handle(j.rw)
		select {
		case j.done <- struct{}{}:
		default:
		}
	}
}

func main() {
	go func() {
		panic(http.ListenAndServe(":8080", nil))
	}()

	r := schedulers.MakeScheduler()
	go testServer(&r)

	listener, err := net.Listen("tcp", "0.0.0.0:6432") // TODO(garet) make this configurable
	if err != nil {
		panic(err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		go func() {
			source := r.NewSource()
			client := frontends.NewClient(conn)
			defer client.Close(nil)
			done := make(chan struct{})
			defer close(done)
			for {
				reader, err := pnet.PreRead(client)
				if err != nil {
					break
				}
				source.Schedule(job{
					rw: pnet.JoinedReadWriter{
						Reader: reader,
						Writer: client,
					},
					done: done,
				}, 0)
				<-done
			}
		}()
	}
}
