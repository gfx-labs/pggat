package routers

import (
	"pggat2/lib/pnet"
	"pggat2/lib/rob"
	"pggat2/lib/router"
)

type Source struct {
	source rob.Source
}

func (T Source) Handle(peer pnet.ReadWriter, write bool) {
	T.source.Schedule(peer, constraints(write))
}

var _ router.Source = Source{}
