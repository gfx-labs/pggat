package interceptor

import (
	"pggat2/lib/middleware"
	"pggat2/lib/util/decorator"
	"pggat2/lib/zap"
)

type Context struct {
	noCopy decorator.NoCopy

	cancelled bool
	zap.ReadWriter
}

func makeContext(rw zap.ReadWriter) Context {
	return Context{
		ReadWriter: rw,
	}
}

func (T *Context) reset() {
	T.cancelled = false
}

func (T *Context) Cancel() {
	T.cancelled = true
}

var _ middleware.Context = (*Context)(nil)
