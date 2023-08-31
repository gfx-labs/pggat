package packets

import "pggat2/lib/fed"

type CopyFail struct {
	Reason string
}

func (T *CopyFail) ReadFromPacket(packet fed.Packet) bool {
	if packet.Type() != TypeCopyFail {
		return false
	}
	packet.ReadString(&T.Reason)
	return true
}

func (T *CopyFail) IntoPacket() fed.Packet {
	packet := fed.NewPacket(TypeCopyFail, len(T.Reason)+1)
	packet = packet.AppendString(T.Reason)
	return packet
}
