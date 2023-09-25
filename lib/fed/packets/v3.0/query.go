package packets

import "gfx.cafe/gfx/pggat/lib/fed"

type Query string

func (T *Query) ReadFromPacket(packet fed.Packet) bool {
	if packet.Type() != TypeQuery {
		return false
	}
	packet.ReadString((*string)(T))
	return true
}

func (T *Query) IntoPacket(packet fed.Packet) fed.Packet {
	packet = packet.Reset(TypeQuery, len(*T)+1)
	packet = packet.AppendString(string(*T))
	return packet
}
