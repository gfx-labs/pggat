package packets

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/util/slices"
)

type AuthenticationSASLFinal []byte

func (T *AuthenticationSASLFinal) ReadFromPacket(packet fed.Packet) bool {
	if packet.Type() != TypeAuthentication {
		return false
	}
	var method int32
	p := packet.ReadInt32(&method)
	if method != 12 {
		return false
	}
	*T = slices.Resize(*T, len(p))
	p.ReadBytes(*T)
	return true
}

func (T *AuthenticationSASLFinal) IntoPacket(packet fed.Packet) fed.Packet {
	packet = packet.Reset(TypeAuthentication, 4+len(*T))
	packet = packet.AppendUint32(12)
	packet = packet.AppendBytes(*T)
	return packet
}
