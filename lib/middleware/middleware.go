package middleware

import "pggat2/lib/zap"

type Middleware interface {
	Read(ctx Context, packet *zap.Packet) error
	ReadUntyped(ctx Context, packet *zap.UntypedPacket) error
	Write(ctx Context, packet *zap.Packet) error
	WriteUntyped(ctx Context, packet *zap.UntypedPacket) error
}
