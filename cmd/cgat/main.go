package main

import (
	"net"
	"net/http"
	_ "net/http/pprof"

	"pggat2/lib/rob/schedulers/v0"
	"pggat2/lib/zap/onebuffer"

	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/bouncer/bouncers/v1"
	"pggat2/lib/bouncer/frontends/v0"
	"pggat2/lib/middleware"
	"pggat2/lib/middleware/interceptor"
	"pggat2/lib/middleware/middlewares/unterminate"
	"pggat2/lib/rob"
	"pggat2/lib/zap"
	"pggat2/lib/zap/zio"
)

type server struct {
	rw zap.ReadWriter
}

func (T server) Do(_ rob.Constraints, work any) {
	client := work.(zap.ReadWriter)
	bouncers.Bounce(client, T.rw)
}

var _ rob.Worker = server{}

func testServer(r rob.Scheduler) {
	conn, err := net.Dial("tcp", "localhost:5432")
	if err != nil {
		panic(err)
	}
	rw := zio.MakeReadWriter(conn)
	backends.Accept(&rw)
	r.AddSink(0, server{
		rw: &rw,
	})
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
			mw := interceptor.MakeInterceptor(&ob, []middleware.Middleware{
				unterminate.Unterminate,
			})
			frontends.Accept(&mw)
			for {
				err = ob.Buffer()
				if err != nil {
					break
				}
				source.Do(0, &mw)
			}
		}()
	}
}
