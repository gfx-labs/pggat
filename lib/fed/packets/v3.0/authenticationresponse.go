package packets

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/util/slices"
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

func (T *AuthenticationResponse) IntoPacket(packet fed.Packet) fed.Packet {
	packet = fed.NewPacket(TypeAuthenticationResponse, len(*T))
	packet = packet.AppendBytes(*T)
	return packet
}
