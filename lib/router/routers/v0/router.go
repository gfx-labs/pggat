package routers

import (
	"pggat2/lib/rob/schedulers/v2"
	"pggat2/lib/router"
)

type Router struct {
	scheduler schedulers.Scheduler
}

func MakeRouter() Router {
	return Router{
		scheduler: schedulers.MakeScheduler(),
	}
}

func NewRouter() *Router {
	r := MakeRouter()
	return &r
}

func (r *Router) NewHandler(write bool) router.Handler {
	sink := r.scheduler.NewSink(constraints(write))
	return MakeHandler(sink)
}

func (r *Router) NewSource() router.Source {
	source := r.scheduler.NewSource()
	return MakeSource(source)
}

var _ router.Router = (*Router)(nil)
