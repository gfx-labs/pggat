package packets

import (
	"gfx.cafe/gfx/pggat/lib/fed"
)

type AuthenticationResponse []byte

func (T *AuthenticationResponse) ReadFrom(packet fed.PacketDecoder) error {
	if packet.Type != TypeAuthenticationResponse {
		return ErrUnexpectedPacket
	}

	return packet.Remaining((*[]byte)(T)).Error
}

func (T *AuthenticationResponse) IntoPacket(packet fed.Packet) fed.Packet {
	packet = fed.NewPacket(TypeAuthenticationResponse, len(*T))
	packet = packet.AppendBytes(*T)
	return packet
}
