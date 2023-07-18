package interceptor

import (
	"pggat2/lib/middleware"
	"pggat2/lib/util/decorator"
	"pggat2/lib/zap"
)

type Context struct {
	noCopy decorator.NoCopy

	cancelled bool

	rw zap.ReadWriter
}

func makeContext(rw zap.ReadWriter) Context {
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

func (T *Context) Write(packet *zap.Packet) error {
	return T.rw.Write(packet)
}

func (T *Context) WriteUntyped(packet *zap.UntypedPacket) error {
	return T.rw.WriteUntyped(packet)
}

var _ middleware.Context = (*Context)(nil)
