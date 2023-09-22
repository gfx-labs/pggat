package packets

import "pggat/lib/fed"

type Close struct {
	Which  byte
	Target string
}

func (T *Close) ReadFromPacket(packet fed.Packet) bool {
	if packet.Type() != TypeClose {
		return false
	}
	p := packet.ReadUint8(&T.Which)
	p = p.ReadString(&T.Target)
	return true
}

func (T *Close) IntoPacket(packet fed.Packet) fed.Packet {
	packet = packet.Reset(TypeClose, 2+len(T.Target))
	packet = packet.AppendUint8(T.Which)
	packet = packet.AppendString(T.Target)
	return packet
}
