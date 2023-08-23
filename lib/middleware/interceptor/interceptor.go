package interceptor

import (
	"crypto/tls"

	"pggat2/lib/middleware"
	"pggat2/lib/zap"
)

type Interceptor struct {
	middlewares []middleware.Middleware
	context     Context
	rw          zap.ReadWriter
}

func NewInterceptor(rw zap.ReadWriter, middlewares ...middleware.Middleware) *Interceptor {
	if v, ok := rw.(*Interceptor); ok {
		v.middlewares = append(v.middlewares, middlewares...)
		return v
	}
	return &Interceptor{
		middlewares: middlewares,
		context:     makeContext(rw),
		rw:          rw,
	}
}

func (T *Interceptor) EnableSSLClient(config *tls.Config) error {
	return T.rw.EnableSSLClient(config)
}

func (T *Interceptor) EnableSSLServer(config *tls.Config) error {
	return T.rw.EnableSSLServer(config)
}

func (T *Interceptor) ReadByte() (byte, error) {
	return T.rw.ReadByte()
}

func (T *Interceptor) Read(packet *zap.Packet) error {
outer:
	for {
		err := T.rw.Read(packet)
		if err != nil {
			return err
		}

		for _, mw := range T.middlewares {
			T.context.reset()
			err := mw.Read(&T.context, packet)
			if err != nil {
				return err
			}
			if T.context.cancelled {
				continue outer
			}
		}

		return nil
	}
}

func (T *Interceptor) ReadUntyped(packet *zap.UntypedPacket) error {
outer:
	for {
		err := T.rw.ReadUntyped(packet)
		if err != nil {
			return err
		}

		for _, mw := range T.middlewares {
			T.context.reset()
			err := mw.ReadUntyped(&T.context, packet)
			if err != nil {
				return err
			}
			if T.context.cancelled {
				continue outer
			}
		}

		return nil
	}
}

func (T *Interceptor) WriteByte(b byte) error {
	return T.rw.WriteByte(b)
}

func (T *Interceptor) Write(packet *zap.Packet) error {
	for _, mw := range T.middlewares {
		T.context.reset()
		err := mw.Write(&T.context, packet)
		if err != nil {
			return err
		}
		if T.context.cancelled {
			return nil
		}
	}

	return T.rw.Write(packet)
}

func (T *Interceptor) WriteUntyped(packet *zap.UntypedPacket) error {
	for _, mw := range T.middlewares {
		T.context.reset()
		err := mw.WriteUntyped(&T.context, packet)
		if err != nil {
			return err
		}
		if T.context.cancelled {
			return nil
		}
	}

	return T.rw.WriteUntyped(packet)
}

func (T *Interceptor) WriteV(packets *zap.Packets) error {
	T.context.packets = packets
	defer func() {
		T.context.packets = nil
	}()
	for T.context.packetsIndex = 0; T.context.packetsIndex < packets.Size(); T.context.packetsIndex++ {
		if packets.IsTyped(T.context.packetsIndex) {
			for _, mw := range T.middlewares {
				T.context.reset()
				err := mw.Write(&T.context, packets.Get(T.context.packetsIndex))
				if err != nil {
					return err
				}
				if T.context.cancelled {
					packets.Remove(T.context.packetsIndex)
					T.context.packetsIndex--
					break
				}
			}
		} else {
			for _, mw := range T.middlewares {
				T.context.reset()
				err := mw.WriteUntyped(&T.context, packets.GetUntyped(T.context.packetsIndex))
				if err != nil {
					return err
				}
				if T.context.cancelled {
					packets.Remove(T.context.packetsIndex)
					T.context.packetsIndex--
					break
				}
			}
		}
	}

	return T.rw.WriteV(packets)
}

func (T *Interceptor) Close() error {
	return T.rw.Close()
}

var _ zap.ReadWriter = (*Interceptor)(nil)
