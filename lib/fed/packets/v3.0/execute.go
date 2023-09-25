package packets

import "gfx.cafe/gfx/pggat/lib/fed"

type Execute struct {
	Target  string
	MaxRows int32
}

func (T *Execute) ReadFromPacket(packet fed.Packet) bool {
	if packet.Type() != TypeExecute {
		return false
	}
	p := packet.ReadString(&T.Target)
	p = p.ReadInt32(&T.MaxRows)
	return true
}

func (T *Execute) IntoPacket(packet fed.Packet) fed.Packet {
	packet = packet.Reset(TypeExecute, len(T.Target)+5)
	packet = packet.AppendString(T.Target)
	packet = packet.AppendInt32(T.MaxRows)
	return packet
}
