package packets

import "gfx.cafe/gfx/pggat/lib/fed"

type Describe struct {
	Which  byte
	Target string
}

func (T *Describe) ReadFromPacket(packet fed.Packet) bool {
	if packet.Type() != TypeDescribe {
		return false
	}
	p := packet.ReadUint8(&T.Which)
	p = p.ReadString(&T.Target)
	return true
}

func (T *Describe) IntoPacket(packet fed.Packet) fed.Packet {
	packet = packet.Reset(TypeDescribe, len(T.Target)+2)
	packet = packet.AppendUint8(T.Which)
	packet = packet.AppendString(T.Target)
	return packet
}
