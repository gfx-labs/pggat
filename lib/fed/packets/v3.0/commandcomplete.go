package packets

import "pggat/lib/fed"

type CommandComplete string

func (T *CommandComplete) ReadFromPacket(packet fed.Packet) bool {
	if packet.Type() != TypeCommandComplete {
		return false
	}
	packet.ReadString((*string)(T))
	return true
}

func (T *CommandComplete) IntoPacket() fed.Packet {
	packet := fed.NewPacket(TypeCommandComplete, len(*T)+1)
	packet = packet.AppendString(string(*T))
	return packet
}
