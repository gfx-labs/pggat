package packets

import "gfx.cafe/gfx/pggat/lib/fed"

type ParameterStatus struct {
	Key   string
	Value string
}

func (T *ParameterStatus) ReadFromPacket(packet fed.Packet) bool {
	if packet.Type() != TypeParameterStatus {
		return false
	}
	p := packet.ReadString(&T.Key)
	p = p.ReadString(&T.Value)
	return true
}

func (T *ParameterStatus) IntoPacket(packet fed.Packet) fed.Packet {
	packet = packet.Reset(TypeParameterStatus, len(T.Key)+len(T.Value)+2)
	packet = packet.AppendString(T.Key)
	packet = packet.AppendString(T.Value)
	return packet
}
