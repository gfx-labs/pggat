package interceptor

import (
	"pggat2/lib/middleware"
	"pggat2/lib/util/decorator"
)

type Context struct {
	noCopy decorator.NoCopy

	cancelled bool
}

func makeContext() Context {
	return Context{}
}

func (T *Context) reset() {
	T.cancelled = false
}

func (T *Context) Cancel() {
	T.cancelled = true
}

var _ middleware.Context = (*Context)(nil)
