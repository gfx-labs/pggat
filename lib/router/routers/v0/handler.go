package routers

import (
	"pggat2/lib/pnet"
	"pggat2/lib/rob"
	"pggat2/lib/router"
)

type Handler struct {
	sink rob.Sink
}

func (T Handler) Next() pnet.ReadWriter {
	return T.sink.Read().(pnet.ReadWriter)
}

var _ router.Handler = Handler{}
