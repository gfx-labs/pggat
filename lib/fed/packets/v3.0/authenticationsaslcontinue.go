package packets

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/util/slices"
)

type AuthenticationSASLContinue []byte

func (T *AuthenticationSASLContinue) ReadFromPacket(packet fed.Packet) bool {
	if packet.Type() != TypeAuthentication {
		return false
	}
	var method int32
	p := packet.ReadInt32(&method)
	if method != 11 {
		return false
	}
	*T = slices.Resize(*T, len(p))
	p.ReadBytes(*T)
	return true
}

func (T *AuthenticationSASLContinue) IntoPacket(packet fed.Packet) fed.Packet {
	packet = packet.Reset(TypeAuthentication, 4+len(*T))
	packet = packet.AppendUint32(11)
	packet = packet.AppendBytes(*T)
	return packet
}
