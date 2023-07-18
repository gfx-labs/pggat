package middleware

import "pggat2/lib/zap"

type Nil struct{}

func (Nil) Read(_ Context, _ *zap.Packet) error {
	return nil
}

func (Nil) ReadUntyped(_ Context, _ *zap.UntypedPacket) error {
	return nil
}

func (Nil) Write(_ Context, _ *zap.Packet) error {
	return nil
}

func (Nil) WriteUntyped(_ Context, _ *zap.UntypedPacket) error {
	return nil
}

var _ Middleware = Nil{}
