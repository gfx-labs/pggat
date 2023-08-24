package packets

import (
	"pggat2/lib/util/slices"
	"pggat2/lib/zap"
)

type AuthenticationResponse []byte

func (T *AuthenticationResponse) ReadFromPacket(packet zap.Packet) bool {
	if packet.Type() != TypeAuthenticationResponse {
		return false
	}
	*T = slices.Resize(*T, len(packet.Payload()))
	packet.ReadBytes(*T)
	return true
}

func (T *AuthenticationResponse) IntoPacket() zap.Packet {
	packet := zap.NewPacket(TypeAuthenticationResponse)
	return packet.AppendBytes(*T)
}
