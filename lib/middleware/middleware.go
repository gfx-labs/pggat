package middleware

import "pggat2/lib/zap"

type Middleware interface {
	Write(ctx Context, packet zap.Inspector) error
	Read(ctx Context, packet zap.Inspector) error
}
