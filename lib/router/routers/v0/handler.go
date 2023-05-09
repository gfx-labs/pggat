package routers

import (
	"pggat2/lib/pnet"
	"pggat2/lib/rob"
	"pggat2/lib/router"
)

type Handler struct {
	sink rob.Sink
}

func MakeHandler(sink rob.Sink) Handler {
	return Handler{
		sink: sink,
	}
}

func (T Handler) Next() pnet.ReadWriter {
	return T.sink.Read().(pnet.ReadWriter)
}

var _ router.Handler = Handler{}
