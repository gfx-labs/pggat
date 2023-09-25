package interceptor

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/middleware"
)

type Interceptor struct {
	middlewares []middleware.Middleware
	context     Context
	rw          fed.Conn
}

func NewInterceptor(rw fed.Conn, middlewares ...middleware.Middleware) *Interceptor {
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

func (T *Interceptor) ReadPacket(typed bool, packet fed.Packet) (fed.Packet, error) {
outer:
	for {
		var err error
		packet, err = T.rw.ReadPacket(typed, packet)
		if err != nil {
			return packet, err
		}

		for _, mw := range T.middlewares {
			T.context.reset()
			err = mw.Read(&T.context, packet)
			if err != nil {
				return packet, err
			}
			if T.context.cancelled {
				continue outer
			}
		}

		return packet, nil
	}
}

func (T *Interceptor) WritePacket(packet fed.Packet) error {
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

var _ fed.Conn = (*Interceptor)(nil)
