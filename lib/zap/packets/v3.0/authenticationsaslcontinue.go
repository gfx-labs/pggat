package packets

import (
	"pggat2/lib/util/slices"
	"pggat2/lib/zap"
)

type AuthenticationSASLContinue []byte

func (T *AuthenticationSASLContinue) ReadFromPacket(packet zap.Packet) bool {
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

func (T *AuthenticationSASLContinue) IntoPacket() zap.Packet {
	packet := zap.NewPacket(TypeAuthentication)
	packet = packet.AppendUint32(11)
	packet = packet.AppendBytes(*T)
	return packet
}
