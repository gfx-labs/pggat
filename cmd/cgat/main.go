package main

import (
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"

	"pggat2/lib/bouncer/bouncers/v2"
	"pggat2/lib/middleware/middlewares/eqp"
	"pggat2/lib/middleware/middlewares/ps"
	"pggat2/lib/rob/schedulers/v1"
	"pggat2/lib/zap/onebuffer"

	"pggat2/lib/bouncer/backends/v0"
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
	psc  *ps.Client
}

type server struct {
	rw   zap.ReadWriter
	eqps *eqp.Server
	pss  *ps.Server
}

func (T server) Do(_ rob.Constraints, w any) {
	job := w.(work)
	job.psc.SetServer(T.pss)
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
	pss := ps.MakeServer()
	mw := interceptor.MakeInterceptor(&rw, []middleware.Middleware{
		&eqps,
		&pss,
	})
	backends.Accept(&mw)
	r.AddSink(0, server{
		rw:   &mw,
		eqps: &eqps,
		pss:  &pss,
	})
}

var DefaultParameterStatus = map[string]string{
	// TODO(garet) we should just get these from the first server connection
	"DateStyle":                     "ISO, MDY",
	"IntervalStyle":                 "postgres",
	"TimeZone":                      "America/Chicago",
	"application_name":              "",
	"client_encoding":               "UTF8",
	"default_transaction_read_only": "off",
	"in_hot_standby":                "off",
	"integer_datetimes":             "on",
	"is_superuser":                  "on",
	"server_encoding":               "UTF8",
	"server_version":                "14.5",
	"session_authorization":         "postgres",
	"standard_conforming_strings":   "on",
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
			defer client.Done()
			ob := onebuffer.MakeOnebuffer(&client)
			eqpc := eqp.MakeClient()
			defer eqpc.Done()
			psc := ps.MakeClient()
			defer psc.Done()
			mw := interceptor.MakeInterceptor(&ob, []middleware.Middleware{
				unterminate.Unterminate,
				&eqpc,
				&psc,
			})
			frontends.Accept(&mw, DefaultParameterStatus)
			for {
				err = ob.Buffer()
				if err != nil {
					break
				}
				source.Do(0, work{
					rw:   &mw,
					eqpc: &eqpc,
					psc:  &psc,
				})
			}
		}()
	}
}
