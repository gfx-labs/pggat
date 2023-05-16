package main

import (
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"

	"pggat2/lib/middleware/middlewares/eqp"
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

type work struct {
	rw   zap.ReadWriter
	eqpc *eqp.Client
}

type server struct {
	rw   zap.ReadWriter
	eqps *eqp.Server
}

func (T server) Do(_ rob.Constraints, w any) {
	job := w.(work)
	T.eqps.SetClient(job.eqpc)
	bouncers.Bounce(job.rw, T.rw)
}

var _ rob.Worker = server{}

func testServer(r rob.Scheduler) {
	conn, err := net.Dial("tcp", "localhost:5432")
	if err != nil {
		panic(err)
	}
	rw := zio.MakeReadWriter(conn)
	eqps := eqp.MakeServer()
	mw := interceptor.MakeInterceptor(&rw, []middleware.Middleware{
		&eqps,
	})
	backends.Accept(&mw)
	r.AddSink(0, server{
		rw:   &mw,
		eqps: &eqps,
	})
}

func main() {
	go func() {
		panic(http.ListenAndServe(":8080", nil))
	}()

	log.Println("Starting pggat...")

	r := schedulers.MakeScheduler()
	for i := 0; i < 5; i++ {
		go testServer(&r)
	}

	listener, err := net.Listen("tcp", "0.0.0.0:6432") // TODO(garet) make this configurable
	if err != nil {
		panic(err)
	}

	log.Println("Listening on 0.0.0.0:6432")

	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		go func() {
			source := r.NewSource()
			client := zio.MakeReadWriter(conn)
			ob := onebuffer.MakeOnebuffer(&client)
			eqpc := eqp.MakeClient()
			mw := interceptor.MakeInterceptor(&ob, []middleware.Middleware{
				unterminate.Unterminate,
				&eqpc,
			})
			frontends.Accept(&mw)
			for {
				err = ob.Buffer()
				if err != nil {
					break
				}
				source.Do(0, work{
					rw:   &mw,
					eqpc: &eqpc,
				})
			}
		}()
	}
}
