package packets

import "pggat2/lib/fed"

type Query string

func (T *Query) ReadFromPacket(packet fed.Packet) bool {
	if packet.Type() != TypeQuery {
		return false
	}
	packet.ReadString((*string)(T))
	return true
}

func (T *Query) IntoPacket() fed.Packet {
	packet := fed.NewPacket(TypeQuery, len(*T)+1)
	packet = packet.AppendString(string(*T))
	return packet
}
