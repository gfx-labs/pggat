package interceptor

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/middleware"
	"gfx.cafe/gfx/pggat/lib/util/decorator"
)

type Context struct {
	noCopy decorator.NoCopy

	cancelled bool

	// for normal Write / WriteUntyped
	rw fed.ReadWriter
}

func makeContext(rw fed.ReadWriter) Context {
	return Context{
		rw: rw,
	}
}

func (T *Context) reset() {
	T.cancelled = false
}

func (T *Context) Cancel() {
	T.cancelled = true
}

func (T *Context) Write(packet fed.Packet) error {
	return T.rw.WritePacket(packet)
}

var _ middleware.Context = (*Context)(nil)
