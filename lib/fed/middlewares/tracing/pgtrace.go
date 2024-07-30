package tracing

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/fed"
)

type packetTrace struct{}

func NewPacketTrace(ctx context.Context) fed.Middleware {
	return &packetTrace{}
}

func (t *packetTrace) ReadPacket(ctx context.Context, packet fed.Packet) (fed.Packet, error) {
	logPacket("Read ", packet)
	return packet, nil
}

func (t *packetTrace) WritePacket(ctx context.Context, packet fed.Packet) (fed.Packet, error) {
	logPacket("Write", packet)
	return packet, nil
}

func (t *packetTrace) PreRead(ctx context.Context, _ bool) (fed.Packet, error) {
	return nil, nil
}

func (t *packetTrace) PostWrite(ctx context.Context) (fed.Packet, error) {
	return nil, nil
}
