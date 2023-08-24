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

func (T *Interceptor) ReadPacket(typed bool) (zap.Packet, error) {
outer:
	for {
		packet, err := T.rw.ReadPacket(typed)
		if err != nil {
			return nil, err
		}

		for _, mw := range T.middlewares {
			T.context.reset()
			err = mw.Read(&T.context, packet)
			if err != nil {
				return nil, err
			}
			if T.context.cancelled {
				continue outer
			}
		}

		return packet, nil
	}
}

func (T *Interceptor) WriteByte(b byte) error {
	return T.rw.WriteByte(b)
}

func (T *Interceptor) WritePacket(packet zap.Packet) error {
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

	return T.rw.WritePacket(packet)
}

func (T *Interceptor) Close() error {
	return T.rw.Close()
}

var _ zap.ReadWriter = (*Interceptor)(nil)
