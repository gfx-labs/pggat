package packets

import "pggat2/lib/fed"

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

func (T *Close) IntoPacket() fed.Packet {
	packet := fed.NewPacket(TypeClose, 2+len(T.Target))
	packet = packet.AppendUint8(T.Which)
	packet = packet.AppendString(T.Target)
	return packet
}
