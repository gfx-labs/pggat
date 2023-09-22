package packets

import (
	"pggat/lib/fed"
)

type ReadyForQuery byte

func (T *ReadyForQuery) ReadFromPacket(packet fed.Packet) bool {
	if packet.Type() != TypeReadyForQuery {
		return false
	}
	packet.ReadUint8((*byte)(T))
	return true
}

func (T *ReadyForQuery) IntoPacket(packet fed.Packet) fed.Packet {
	packet = fed.NewPacket(TypeReadyForQuery, 1)
	packet = packet.AppendUint8(byte(*T))
	return packet
}
