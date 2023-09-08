package packets

import (
	"pggat/lib/fed"
	"pggat/lib/util/slices"
)

type AuthenticationResponse []byte

func (T *AuthenticationResponse) ReadFromPacket(packet fed.Packet) bool {
	if packet.Type() != TypeAuthenticationResponse {
		return false
	}
	*T = slices.Resize(*T, len(packet.Payload()))
	packet.ReadBytes(*T)
	return true
}

func (T *AuthenticationResponse) IntoPacket() fed.Packet {
	packet := fed.NewPacket(TypeAuthenticationResponse, len(*T))
	return packet.AppendBytes(*T)
}
