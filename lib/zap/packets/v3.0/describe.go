package packets

import "pggat2/lib/zap"

type Describe struct {
	Which  byte
	Target string
}

func (T *Describe) ReadFromPacket(packet zap.Packet) bool {
	if packet.Type() != TypeDescribe {
		return false
	}
	p := packet.ReadUint8(&T.Which)
	p = p.ReadString(&T.Target)
	return true
}

func (T *Describe) IntoPacket() zap.Packet {
	packet := zap.NewPacket(TypeDescribe)
	packet = packet.AppendUint8(T.Which)
	packet = packet.AppendString(T.Target)
	return packet
}
