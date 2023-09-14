package packets

import (
	"pggat/lib/fed"
	"pggat/lib/util/slices"
)

type CopyData []byte

func (T *CopyData) ReadFromPacket(packet fed.Packet) bool {
	if packet.Type() != TypeCopyData {
		return false
	}

	*T = slices.Resize(*T, len(packet.Payload()))
	packet.ReadBytes(*T)
	return true
}

func (T *CopyData) IntoPacket() fed.Packet {
	packet := fed.NewPacket(TypeCopyData, len(*T))
	packet = packet.AppendBytes(*T)
	return packet
}
