package packets

import "gfx.cafe/gfx/pggat/lib/fed"

type CommandComplete string

func (T *CommandComplete) ReadFromPacket(packet fed.Packet) bool {
	if packet.Type() != TypeCommandComplete {
		return false
	}
	packet.ReadString((*string)(T))
	return true
}

func (T *CommandComplete) IntoPacket(packet fed.Packet) fed.Packet {
	packet = packet.Reset(TypeCommandComplete, len(*T)+1)
	packet = packet.AppendString(string(*T))
	return packet
}
