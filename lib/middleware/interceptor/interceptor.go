package interceptor

import (
	"time"

	"pggat2/lib/middleware"
	"pggat2/lib/zap"
)

type Interceptor struct {
	middlewares []middleware.Middleware
	context     Context
	rw          zap.ReadWriter
}

func NewInterceptor(rw zap.ReadWriter, middlewares ...middleware.Middleware) *Interceptor {
	return &Interceptor{
		middlewares: middlewares,
		context:     makeContext(),
	}
}

func (T *Interceptor) ReadInto(buffer *zap.Buffer, typed bool) error {
	pre := buffer.Count()

	if err := T.rw.ReadInto(buffer, typed); err != nil {
		return err
	}

	post := buffer.Count()

	for i := pre; i < post; i++ {
		for _, mw := range T.middlewares {
			T.context.reset()
			if err := mw.Read(&T.context, buffer.Inspect(i)); err != nil {
				return err
			}

			if T.context.cancelled {
				// TODO(garet) cancel packet
				panic("TODO")
			}
		}
	}

	return nil
}

func (T *Interceptor) SetReadDeadline(time time.Time) error {
	return T.rw.SetReadDeadline(time)
}

func (T *Interceptor) WriteFrom(buffer *zap.Buffer) error {
	for i := 0; i < buffer.Count(); i++ {
		for _, mw := range T.middlewares {
			T.context.reset()
			if err := mw.Write(&T.context, buffer.Inspect(i)); err != nil {
				return err
			}

			if T.context.cancelled {
				// TODO(garet) cancel packet
				panic("TODO")
			}
		}
	}

	return T.rw.WriteFrom(buffer)
}

func (T *Interceptor) SetWriteDeadline(time time.Time) error {
	return T.rw.SetWriteDeadline(time)
}

var _ zap.ReadWriter = (*Interceptor)(nil)
