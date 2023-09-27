package interceptor

import (
	"net"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/middleware"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type Interceptor struct {
	middlewares []middleware.Middleware
	context     Context
	conn        fed.Conn
}

func NewInterceptor(conn fed.Conn, middlewares ...middleware.Middleware) *Interceptor {
	if v, ok := conn.(*Interceptor); ok {
		v.middlewares = append(v.middlewares, middlewares...)
		return v
	}
	return &Interceptor{
		middlewares: middlewares,
		context:     makeContext(conn),
		conn:        conn,
	}
}

func (T *Interceptor) ReadPacket(typed bool, packet fed.Packet) (fed.Packet, error) {
outer:
	for {
		var err error
		packet, err = T.conn.ReadPacket(typed, packet)
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

	return T.conn.WritePacket(packet)
}

func (T *Interceptor) LocalAddr() net.Addr {
	return T.conn.LocalAddr()
}

func (T *Interceptor) RemoteAddr() net.Addr {
	return T.conn.RemoteAddr()
}

func (T *Interceptor) SSLEnabled() bool {
	return T.conn.SSLEnabled()
}

func (T *Interceptor) User() string {
	return T.conn.User()
}

func (T *Interceptor) Database() string {
	return T.conn.Database()
}

func (T *Interceptor) InitialParameters() map[strutil.CIString]string {
	return T.conn.InitialParameters()
}

func (T *Interceptor) Close() error {
	return T.conn.Close()
}

var _ fed.Conn = (*Interceptor)(nil)
