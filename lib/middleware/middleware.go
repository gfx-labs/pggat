package middleware

import "pggat2/lib/zap"

type Middleware interface {
	Read(ctx Context, packet zap.Packet) error
	Write(ctx Context, packet zap.Packet) error
}
