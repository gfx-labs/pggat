package packets

import "pggat2/lib/zap"

type CopyFail struct {
	Reason string
}

func (T *CopyFail) ReadFromPacket(packet zap.Packet) bool {
	if packet.Type() != TypeCopyFail {
		return false
	}
	packet.ReadString(&T.Reason)
	return true
}

func (T *CopyFail) IntoPacket() zap.Packet {
	packet := zap.NewPacket(TypeCopyFail)
	packet = packet.AppendString(T.Reason)
	return packet
}
