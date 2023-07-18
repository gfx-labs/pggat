package interceptor

import (
	"pggat2/lib/middleware"
	"pggat2/lib/util/decorator"
	"pggat2/lib/zap"
)

type Context struct {
	noCopy decorator.NoCopy

	cancelled bool

	// for normal Write / WriteUntyped
	rw zap.ReadWriter

	// for Write / WriteUntyped into packets
	packets      *zap.Packets
	packetsIndex int
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
	if T.packets != nil {
		cloned := zap.NewPacket()
		cloned.WriteType(packet.ReadType())
		cloned.WriteBytes(packet.Payload())
		T.packets.InsertBefore(T.packetsIndex, cloned)
		return nil
	}
	return T.rw.Write(packet)
}

func (T *Context) WriteUntyped(packet *zap.UntypedPacket) error {
	if T.packets != nil {
		cloned := zap.NewUntypedPacket()
		cloned.WriteBytes(packet.Payload())
		T.packets.InsertUntypedBefore(T.packetsIndex, cloned)
		return nil
	}
	return T.rw.WriteUntyped(packet)
}

var _ middleware.Context = (*Context)(nil)
