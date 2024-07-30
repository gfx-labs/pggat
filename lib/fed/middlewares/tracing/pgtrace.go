package tracing

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/fed"
)

type pgtrace struct{}

func NewPgTrace(ctx context.Context) fed.Middleware {
	return &pgtrace{}
}

func (t *pgtrace) ReadPacket(ctx context.Context, packet fed.Packet) (fed.Packet, error) {
	logPacket("ReadPacket ", packet)
	return packet, nil
}

func (t *pgtrace) WritePacket(ctx context.Context, packet fed.Packet) (fed.Packet, error) {
	logPacket("WritePacket", packet)
	return packet, nil
}

func (t *pgtrace) PreRead(ctx context.Context, _ bool) (fed.Packet, error) {
	return nil, nil
}

func (t *pgtrace) PostWrite(ctx context.Context) (fed.Packet, error) {
	return nil, nil
}
