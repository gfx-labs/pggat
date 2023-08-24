package packets

import "pggat2/lib/zap"

type Query string

func (T *Query) ReadFromPacket(packet zap.Packet) bool {
	if packet.Type() != TypeQuery {
		return false
	}
	packet.ReadString((*string)(T))
	return true
}

func (T *Query) IntoPacket() zap.Packet {
	packet := zap.NewPacket(TypeQuery, len(*T)+1)
	packet = packet.AppendString(string(*T))
	return packet
}
