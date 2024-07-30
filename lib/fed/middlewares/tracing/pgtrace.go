package tracing

import (
	"gfx.cafe/gfx/pggat/lib/fed"
)

type packetTrace struct{}

func NewPacketTrace() fed.Middleware {
	return &packetTrace{}
}

func (t *packetTrace) ReadPacket(packet fed.Packet) (fed.Packet, error) {
	logPacket("Read ", packet)
	return packet, nil
}

func (t *packetTrace) WritePacket(packet fed.Packet) (fed.Packet, error) {
	logPacket("Write", packet)
	return packet, nil
}

func (t *packetTrace) PreRead(_ bool) (fed.Packet, error) {
	return nil, nil
}

func (t *packetTrace) PostWrite() (fed.Packet, error) {
	return nil, nil
}
