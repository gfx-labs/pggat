package packets

import "pggat2/lib/zap"

type Execute struct {
	Target  string
	MaxRows int32
}

func (T *Execute) ReadFromPacket(packet zap.Packet) bool {
	if packet.Type() != TypeExecute {
		return false
	}
	p := packet.ReadString(&T.Target)
	p = p.ReadInt32(&T.MaxRows)
	return true
}

func (T *Execute) IntoPacket() zap.Packet {
	packet := zap.NewPacket(TypeExecute, len(T.Target)+5)
	packet = packet.AppendString(T.Target)
	packet = packet.AppendInt32(T.MaxRows)
	return packet
}
