package packets

import (
	"pggat2/lib/util/slices"
	"pggat2/lib/zap"
)

type AuthenticationSASLFinal []byte

func (T *AuthenticationSASLFinal) ReadFromPacket(packet zap.Packet) bool {
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

func (T *AuthenticationSASLFinal) IntoPacket() zap.Packet {
	packet := zap.NewPacket(TypeAuthentication, 4+len(*T))
	packet = packet.AppendUint32(12)
	packet = packet.AppendBytes(*T)
	return packet
}
