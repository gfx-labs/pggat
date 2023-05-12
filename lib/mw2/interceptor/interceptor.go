package interceptor

import (
	"pggat2/lib/mw2"
	"pggat2/lib/util/decorator"
	"pggat2/lib/zap"
)

type Interceptor struct {
	noCopy decorator.NoCopy

	middlewares []mw2.Middleware
	Context
}

func MakeInterceptor(rw zap.ReadWriter, middlewares []mw2.Middleware) Interceptor {
	return Interceptor{
		middlewares: middlewares,
		Context:     makeContext(rw),
	}
}

func (T *Interceptor) Read() (zap.In, error) {
	for {
		in, err := T.ReadWriter.Read()
		if err != nil {
			return zap.In{}, err
		}

		T.Context.reset()
		for _, mw := range T.middlewares {
			err = mw.Read(&T.Context, in)
			if err != nil {
				return zap.In{}, err
			}
			if T.cancelled {
				break
			}
		}

		if !T.cancelled {
			return in, nil
		}
	}
}

func (T *Interceptor) ReadUntyped() (zap.In, error) {
	for {
		in, err := T.ReadWriter.ReadUntyped()
		if err != nil {
			return zap.In{}, err
		}

		T.Context.reset()
		for _, mw := range T.middlewares {
			err = mw.Read(&T.Context, in)
			if err != nil {
				return zap.In{}, err
			}
			if T.cancelled {
				break
			}
		}

		if !T.cancelled {
			return in, nil
		}
	}
}

func (T *Interceptor) Send(out zap.Out) error {
	T.Context.reset()
	for _, mw := range T.middlewares {
		err := mw.Send(&T.Context, out)
		if err != nil {
			return err
		}
		if T.cancelled {
			break
		}
	}

	if !T.cancelled {
		return T.Context.ReadWriter.Send(out)
	}
	return nil
}

var _ zap.ReadWriter = (*Interceptor)(nil)
