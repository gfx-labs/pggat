package packets

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/util/slices"
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

func (T *CopyData) IntoPacket(packet fed.Packet) fed.Packet {
	packet = fed.NewPacket(TypeCopyData, len(*T))
	packet = packet.AppendBytes(*T)
	return packet
}
