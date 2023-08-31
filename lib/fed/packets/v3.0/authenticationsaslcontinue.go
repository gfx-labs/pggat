package packets

import (
	"pggat2/lib/fed"
	"pggat2/lib/util/slices"
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

func (T *AuthenticationSASLContinue) IntoPacket() fed.Packet {
	packet := fed.NewPacket(TypeAuthentication, 4+len(*T))
	packet = packet.AppendUint32(11)
	packet = packet.AppendBytes(*T)
	return packet
}
