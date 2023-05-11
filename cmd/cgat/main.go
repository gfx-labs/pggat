package main

import (
	"net"
	"net/http"
	_ "net/http/pprof"

	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/bouncer/bouncers/v0"
	"pggat2/lib/bouncer/frontends/v0"
	"pggat2/lib/middleware/middlewares/eqp"
	"pggat2/lib/middleware/middlewares/unread"
	"pggat2/lib/middleware/middlewares/unterminate"
	"pggat2/lib/pnet"
	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v2"
)

type job struct {
	client pnet.ReadWriter
	done   chan<- struct{}
}

func testServer(r rob.Scheduler) {
	conn, err := net.Dial("tcp", "localhost:5432")
	if err != nil {
		panic(err)
	}
	server := pnet.MakeIOReadWriter(conn)
	backends.Accept(&server)
	consumer := eqp.MakeConsumer(&server)
	sink := r.NewSink(0)
	for {
		j := sink.Read().(job)
		bouncers.Bounce(j.client, &consumer)
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
			client := pnet.MakeIOReadWriter(conn)
			ut := unterminate.MakeUnterminate(&client)
			frontends.Accept(ut)
			creator := eqp.MakeCreator(ut)
			done := make(chan struct{})
			defer close(done)
			for {
				u, err := unread.NewUnread(&creator)
				if err != nil {
					break
				}
				source.Schedule(job{
					client: u,
					done:   done,
				}, 0)
				<-done
			}
		}()
	}
}
