package main

import (
	"net"
	"net/http"
	_ "net/http/pprof"

	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/bouncer/bouncers/v1"
	"pggat2/lib/bouncer/frontends/v0"
	"pggat2/lib/middleware/middlewares/onebuffer"
	"pggat2/lib/mw2"
	"pggat2/lib/mw2/interceptor"
	"pggat2/lib/mw2/middlewares/unterminate"
	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v2"
	"pggat2/lib/zap"
	"pggat2/lib/zap/zio"
)

type job struct {
	client zap.ReadWriter
	done   chan<- struct{}
}

func testServer(r rob.Scheduler) {
	conn, err := net.Dial("tcp", "localhost:5432")
	if err != nil {
		panic(err)
	}
	server := zio.MakeReadWriter(conn)
	backends.Accept(&server)
	sink := r.NewSink(0)
	for {
		j := sink.Read().(job)
		bouncers.Bounce(j.client, &server)
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
			client := zio.MakeReadWriter(conn)
			ob := onebuffer.MakeOnebuffer(&client)
			mw := interceptor.MakeInterceptor(&ob, []mw2.Middleware{
				unterminate.Unterminate,
			})
			frontends.Accept(&mw)
			done := make(chan struct{})
			defer close(done)
			for {
				err = ob.Buffer()
				if err != nil {
					break
				}
				source.Schedule(job{
					client: &mw,
					done:   done,
				}, 0)
				<-done
			}
		}()
	}
}
