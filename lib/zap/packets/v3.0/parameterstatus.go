package packets

import "pggat2/lib/zap"

type ParameterStatus struct {
	Key   string
	Value string
}

func (T *ParameterStatus) ReadFromPacket(packet zap.Packet) bool {
	if packet.Type() != TypeParameterStatus {
		return false
	}
	p := packet.ReadString(&T.Key)
	p = p.ReadString(&T.Value)
	return true
}

func (T *ParameterStatus) IntoPacket() zap.Packet {
	packet := zap.NewPacket(TypeParameterStatus, len(T.Key)+len(T.Value)+2)
	packet = packet.AppendString(T.Key)
	packet = packet.AppendString(T.Value)
	return packet
}
