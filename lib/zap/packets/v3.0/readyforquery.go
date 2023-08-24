package packets

import (
	"pggat2/lib/zap"
)

type ReadyForQuery byte

func (T *ReadyForQuery) ReadFromPacket(packet zap.Packet) bool {
	if packet.Type() != TypeReadyForQuery {
		return false
	}
	packet.ReadUint8((*byte)(T))
	return true
}

func (T *ReadyForQuery) IntoPacket() zap.Packet {
	packet := zap.NewPacket(TypeReadyForQuery, 1)
	packet = packet.AppendUint8(byte(*T))
	return packet
}
