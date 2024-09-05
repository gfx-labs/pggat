package tracing

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/fed"
)

type packetTrace struct{}

func NewPacketTrace() fed.Middleware {
	return &packetTrace{}
}

func (t *packetTrace) ReadPacket(_ context.Context, packet fed.Packet) (fed.Packet, error) {
	logPacket("Read ", packet)
	return packet, nil
}

func (t *packetTrace) WritePacket(_ context.Context, packet fed.Packet) (fed.Packet, error) {
	logPacket("Write", packet)
	return packet, nil
}

func (t *packetTrace) PreRead(_ context.Context, _ bool) (fed.Packet, error) {
	return nil, nil
}

func (t *packetTrace) PostWrite(_ context.Context) (fed.Packet, error) {
	return nil, nil
}
