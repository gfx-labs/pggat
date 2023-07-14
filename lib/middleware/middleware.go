package middleware

import "pggat2/lib/zap"

type Middleware interface {
	Send(ctx Context, out zap.Out) error
	Read(ctx Context, in zap.In) error
}
